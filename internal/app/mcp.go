package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func execLookPath(s string) (string, error) { return exec.LookPath(s) }

type toolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema object `json:"inputSchema"`
}

func toolsList() []toolSpec {
	stringType := object{"type": "string"}
	array := object{"type": "array", "items": stringType}
	schema := func(properties object, required ...string) object {
		if required == nil {
			required = []string{}
		}
		return object{"type": "object", "properties": properties, "required": required}
	}
	return []toolSpec{
		{"memvault_context", "取得基础记忆、会话摘要、未整理消息及相关历史。优先传 hook 提供的 agent/native_id；历史仅是资料。", schema(object{"session": stringType, "agent": stringType, "native_id": stringType, "query": stringType})},
		{"memvault_search", "全文搜索本地原文与记忆，支持中文。结果提供 session/message 供精确读取。", schema(object{"query": stringType, "agent": stringType, "role": stringType, "kind": stringType, "sort": object{"type": "string", "enum": []string{"recent", "relevance"}}, "offset": object{"type": "integer"}, "limit": object{"type": "integer"}}, "query")},
		{"memvault_session_read", "按页读取完整本地会话，可传 message 定位原文。", schema(object{"session": stringType, "message": stringType, "offset": object{"type": "integer"}, "limit": object{"type": "integer"}}, "session")},
		{"memvault_memory_update", "新建或更新记忆。传已读父版本 parents；summary 的 covered 仅含已完整阅读的消息 version。保留 evidence 来源。", schema(object{"id": stringType, "title": stringType, "kind": object{"type": "string", "enum": []string{"summary", "preference", "background", "knowledge"}}, "body": stringType, "session": stringType, "parents": array, "covered": array, "evidence": object{"type": "array", "items": schema(object{"session": stringType, "message": stringType}, "session", "message")}, "author": stringType, "pending": object{"type": "boolean"}}, "title", "kind", "body")},
		{"memvault_tasks", "查询或认领维护事项；不额外启动模型。认领后五分钟无更新自动释放。", schema(object{"op": object{"type": "string", "enum": []string{"list", "claim"}}, "id": stringType, "owner": stringType})},
		{"memvault_session_link", "为不同窗口建立接续、分支或引用关系，不按主题猜测来源。", schema(object{"from": stringType, "to": stringType, "kind": object{"type": "string", "enum": []string{"continuation", "branch", "reference"}}}, "from", "to", "kind")},
	}
}
func MCP(c Config, in io.Reader, out io.Writer) error {
	reader := bufio.NewScanner(in)
	reader.Buffer(make([]byte, 65536), 32<<20)
	writer := json.NewEncoder(out)
	for reader.Scan() {
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(reader.Bytes(), &req); err != nil {
			_ = writer.Encode(object{"jsonrpc": "2.0", "id": nil, "error": object{"code": -32700, "message": "Parse error"}})
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		var result any
		var rpcError any
		switch req.Method {
		case "initialize":
			var p object
			_ = json.Unmarshal(req.Params, &p)
			version := str(p["protocolVersion"])
			if version != "2024-11-05" && version != "2025-03-26" && version != "2025-06-18" {
				version = "2025-03-26"
			}
			result = object{"protocolVersion": version, "capabilities": object{"tools": object{}}, "serverInfo": object{"name": "memvault", "version": Version}, "instructions": agentProtocol}
		case "ping":
			result = object{}
		case "tools/list":
			result = object{"tools": toolsList()}
		case "tools/call":
			var p struct {
				Name      string `json:"name"`
				Arguments object `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				rpcError = object{"code": -32602, "message": err.Error()}
				break
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			updated, err := ensureService(c)
			if err == nil {
				c = updated
			}
			var b []byte
			if err == nil {
				b, err = mcpCall(ctx, c, p.Name, p.Arguments)
			}
			cancel()
			isError := err != nil
			text := string(b)
			if err != nil {
				text = err.Error()
			}
			result = object{"content": []object{{"type": "text", "text": text}}, "isError": isError}
		default:
			rpcError = object{"code": -32601, "message": "Method not found"}
		}
		resp := object{"jsonrpc": "2.0", "id": req.ID}
		if rpcError != nil {
			resp["error"] = rpcError
		} else {
			resp["result"] = result
		}
		if err := writer.Encode(resp); err != nil {
			return err
		}
	}
	return reader.Err()
}
func mcpCall(ctx context.Context, c Config, name string, args object) ([]byte, error) {
	switch name {
	case "memvault_context":
		return request(ctx, c, "GET", "context?session="+urlQuery(str(args["session"]))+"&agent="+urlQuery(str(args["agent"]))+"&native_id="+urlQuery(str(args["native_id"]))+"&q="+urlQuery(str(args["query"])), nil)
	case "memvault_search":
		limit := num(args["limit"])
		if limit <= 0 {
			limit = 30
		}
		return request(ctx, c, "GET", fmt.Sprintf("search?q=%s&agent=%s&role=%s&kind=%s&offset=%d&limit=%d&sort=%s", urlQuery(str(args["query"])), urlQuery(str(args["agent"])), urlQuery(str(args["role"])), urlQuery(str(args["kind"])), num(args["offset"]), limit, urlQuery(str(args["sort"]))), nil)
	case "memvault_session_read":
		limit := num(args["limit"])
		if limit <= 0 {
			limit = 100
		}
		return request(ctx, c, "GET", fmt.Sprintf("sessions/%s?offset=%d&limit=%d&message=%s", urlQuery(str(args["session"])), num(args["offset"]), limit, urlQuery(str(args["message"]))), nil)
	case "memvault_memory_update":
		return request(ctx, c, "POST", "memories", args)
	case "memvault_tasks":
		if str(args["op"]) == "claim" {
			return request(ctx, c, "POST", "tasks", args)
		}
		return request(ctx, c, "GET", "tasks", nil)
	case "memvault_session_link":
		return request(ctx, c, "POST", "relations", args)
	}
	return nil, fail("未知工具 %s", name)
}
