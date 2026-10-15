package app

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/klauspost/compress/zstd"
)

type object = map[string]any

func obj(v any) object {
	m, _ := v.(map[string]any)
	if m == nil {
		return object{}
	}
	return m
}
func str(v any) string { s, _ := v.(string); return s }
func num(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		x, _ := n.Int64()
		return x
	case int64:
		return n
	}
	return 0
}
func arr(v any) []any { a, _ := v.([]any); return a }
func stamp(v any) string {
	if s := str(v); s != "" {
		return s
	}
	n := num(v)
	if n == 0 {
		return ""
	}
	if n > 1e11 {
		return time.UnixMilli(n).UTC().Format(time.RFC3339Nano)
	}
	return time.Unix(n, 0).UTC().Format(time.RFC3339Nano)
}
func first(values ...string) string {
	for _, s := range values {
		if s != "" {
			return s
		}
	}
	return ""
}
func (e *Engine) attachment(p object) Attachment {
	name := filepath.Base(first(str(p["filename"]), str(p["name"]), str(p["path"]), "附件"))
	a := Attachment{Name: name, Missing: true}
	source := obj(p["source"])
	u := first(str(p["url"]), str(p["image_url"]), str(obj(p["image_url"])["url"]), str(source["url"]))
	path := str(p["path"])
	var data []byte
	var err error
	if strings.HasPrefix(u, "file://") {
		parsed, _ := url.Parse(u)
		path = parsed.Path
		if parsed.Host != "" && parsed.Host != "localhost" {
			path = ""
		}
	}
	if path != "" {
		data, err = os.ReadFile(path)
		a.Name = filepath.Base(path)
	} else if strings.HasPrefix(u, "data:") && strings.Contains(u, ";base64,") {
		data, err = base64.StdEncoding.DecodeString(strings.SplitN(u, ";base64,", 2)[1])
	} else if str(source["type"]) == "base64" {
		data, err = base64.StdEncoding.DecodeString(str(source["data"]))
	} else {
		a.URL = u
		return a
	}
	if err != nil {
		return a
	}
	id, err := e.Blob(data)
	if err != nil {
		return a
	}
	a.Blob = id
	a.Size = int64(len(data))
	a.Missing = false
	return a
}
func (e *Engine) parts(v any) (string, []Attachment) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	var text []string
	var attachments []Attachment
	for _, x := range arr(v) {
		p := obj(x)
		switch str(p["type"]) {
		case "text", "input_text", "output_text":
			text = append(text, str(p["text"]))
		case "image", "input_image", "document", "file":
			attachments = append(attachments, e.attachment(p))
		case "tool_result":
			if t, ok := p["content"].(string); ok {
				text = append(text, "工具结果：\n"+t)
			}
		}
	}
	return strings.Join(text, "\n"), attachments
}

type RawManifest struct {
	Format   string   `json:"format"`
	Bytes    int64    `json:"bytes"`
	Chunks   []string `json:"chunks"`
	Complete bool     `json:"complete"`
}

func (e *Engine) ArchiveRaw(r io.Reader, format string) (string, error) {
	m := RawManifest{Format: format, Complete: true}
	buf := make([]byte, 256<<10)
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			id, er := e.Blob(buf[:n])
			if er != nil {
				return "", er
			}
			m.Chunks = append(m.Chunks, id)
			m.Bytes += int64(n)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return e.Blob(encoded(m))
}
func (e *Engine) ParseTranscript(agent string, data []byte, fallback string) (Session, error) {
	return e.parseTranscript(agent, bytes.NewReader(data), fallback)
}
func (e *Engine) parseTranscript(agent string, reader io.Reader, fallback string) (Session, error) {
	s := Session{Agent: agent, NativeID: fallback, Coverage: "来源中可取得的可见消息；不代表平台全部历史"}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 65536), 128<<20)
	line := int64(0)
	turnID := ""
	turnCount := int64(0)
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		var row object
		if err := json.Unmarshal(raw, &row); err != nil {
			s.Issues = append(s.Issues, fmt.Sprintf("第 %d 行待补齐或无法解析", line))
			continue
		}
		var m Message
		var present bool
		if agent == "codex" {
			p := obj(row["payload"])
			if (str(row["type"]) == "event_msg" && str(p["type"]) == "task_started") || str(row["type"]) == "turn_context" {
				if id := str(p["turn_id"]); id != "" && id != turnID {
					turnID = id
					turnCount = 0
				}
			}
			if str(row["type"]) == "session_meta" {
				s.NativeID = first(str(p["id"]), str(p["session_id"]), s.NativeID)
				s.Created = first(str(p["timestamp"]), str(row["timestamp"]))
				s.Parent = first(str(p["forked_from_id"]), str(p["parent_thread_id"]), str(obj(obj(obj(p["source"])["subagent"])["thread_spawn"])["parent_thread_id"]))
			}
			if str(row["type"]) == "retained_context" && str(p["type"]) == "verified_answer" {
				m = Message{ID: "answer:" + str(p["turn_id"]) + ":" + str(p["call_id"]), Role: "user", Text: string(encoded(p["questions"])), Order: line, Revision: sourceRevision(str(row["timestamp"]), line), Time: str(row["timestamp"]), Complete: true}
				present = true
			}
			if str(row["type"]) == "response_item" && str(p["type"]) == "message" {
				role := str(p["role"])
				if (role != "user" && role != "assistant") || str(p["channel"]) == "analysis" {
					continue
				}
				meta := obj(p["internal_chat_message_metadata_passthrough"])
				kinds := arr(meta["content_item_kinds"])
				if role == "user" && len(kinds) > 0 {
					human := false
					for _, k := range kinds {
						if k == "user.text" || k == "unknown" {
							human = true
						}
					}
					if !human {
						continue
					}
				}
				text, attachments := e.parts(p["content"])
				if strings.HasPrefix(text, "<environment_context>") || strings.HasPrefix(text, "<external_codex_apps_open_page>") {
					continue
				}
				turnCount++
				fallbackID := fmt.Sprintf("line:%d", line)
				if turnID != "" {
					fallbackID = fmt.Sprintf("turn:%s:%d", turnID, turnCount)
				}
				m = Message{ID: first(str(p["id"]), fallbackID), Role: role, Text: text, Order: line, Revision: sourceRevision(str(row["timestamp"]), line), Time: first(stamp(meta["create_time"]), str(row["timestamp"])), Complete: true, Phase: str(p["phase"]), Attachments: attachments}
				present = true
			}
		} else {
			if strings.HasPrefix(fallback, "agent-") {
				s.NativeID = first(str(row["sessionId"]), "unknown") + ":" + fallback
				s.Parent = str(row["sessionId"])
			} else {
				s.NativeID = first(str(row["sessionId"]), s.NativeID)
			}
			p := obj(row["message"])
			role := first(str(p["role"]), str(row["type"]))
			if (role != "user" && role != "assistant") || row["isMeta"] == true || row["isCompactSummary"] == true {
				continue
			}
			text, attachments := e.parts(p["content"])
			if text == "" && len(attachments) == 0 {
				continue
			}
			if strings.HasPrefix(text, "<local-command-") || strings.HasPrefix(text, "<command-name>") {
				continue
			}
			m = Message{ID: first(str(row["uuid"]), str(p["id"]), fmt.Sprintf("line:%d", line)), Role: role, Text: text, Order: line, Revision: sourceRevision(str(row["timestamp"]), line), Time: str(row["timestamp"]), Complete: true, Attachments: attachments}
			present = true
		}
		if present {
			if s.Created == "" {
				s.Created = m.Time
			}
			if m.Time > s.Updated {
				s.Updated = m.Time
			}
			if m.Role == "user" && s.Title == "" {
				r := []rune(strings.Split(m.Text, "\n")[0])
				if len(r) > 72 {
					r = r[:72]
				}
				s.Title = string(r)
			}
			s.Messages = append(s.Messages, m)
		}
	}
	if err := scanner.Err(); err != nil {
		s.Issues = append(s.Issues, "正文解析中断："+err.Error())
	}
	if s.NativeID == "" {
		s.NativeID = hash(encoded(s.Messages))[:32]
	}
	s.ID = key(agent, s.NativeID)
	if s.Title == "" {
		s.Title = agent + " · " + s.NativeID
	}

	if s.Parent != "" {
		s.Parent = key(agent, s.Parent)
	}
	s.Count = len(s.Messages)
	return s, nil
}
func (e *Engine) captureSession(s Session) error {
	if e.Forgotten("session", s.ID) {
		return nil
	}
	var events []Event
	meta := s
	meta.Messages = nil
	meta.Count = 0
	meta.Missing = 0
	for _, m := range s.Messages {
		for _, a := range m.Attachments {
			if a.Missing {
				meta.Missing++
			}
		}
	}
	events = append(events, newEvent(e.Config.Device, "session", meta))
	for _, m := range s.Messages {
		events = append(events, newEvent(e.Config.Device, "message", Change{Session: s.ID, Message: &m}))
	}
	for _, m := range s.Messages {
		if m.Role == "user" {
			match := regexp.MustCompile(`^\[memvault-continue:([a-f0-9]{32})\]`).FindStringSubmatch(m.Text)
			if len(match) == 2 && match[1] != s.ID {
				events = append(events, newEvent(e.Config.Device, "relation", Relation{s.ID, match[1], "continuation"}))
				break
			}
		}
	}
	if s.Parent != "" {
		events = append(events, newEvent(e.Config.Device, "relation", Relation{s.ID, s.Parent, "branch"}))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.persist(events); err != nil {
		return err
	}
	return nil
}
func (e *Engine) CaptureFile(agent, path string, force bool) error {
	e.assetMu.Lock()
	defer e.assetMu.Unlock()
	if _, err := os.Stat(filepath.Join(e.Config.Root, "memvault.json")); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	var size, mtime int64
	_ = e.DB.QueryRow("SELECT size,mtime FROM cursors WHERE path=?", path).Scan(&size, &mtime)
	if !force && size == info.Size() && mtime == info.ModTime().UnixNano() {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	snapshot, err := os.CreateTemp(stateDir(), "source-snapshot-*")
	if err != nil {
		return err
	}
	defer snapshot.Close()
	defer os.Remove(snapshot.Name())
	raw, err := e.ArchiveRaw(io.TeeReader(io.LimitReader(f, info.Size()), snapshot), "native-"+agent)
	if err != nil {
		return err
	}
	if _, err = snapshot.Seek(0, 0); err != nil {
		return err
	}
	var reader io.Reader = snapshot
	var dec *zstd.Decoder
	if strings.HasSuffix(path, ".zst") {
		dec, err = zstd.NewReader(snapshot, zstd.WithDecoderMaxMemory(1<<30))
		if err != nil {
			return err
		}
		defer dec.Close()
		reader = dec
	}
	s, err := e.parseTranscript(agent, reader, strings.TrimSuffix(strings.TrimSuffix(filepath.Base(path), ".zst"), ".jsonl"))
	if err != nil {
		return err
	}
	s.Raw = []string{raw}
	if e.Forgotten("session", s.ID) {
		e.mu.Lock()
		markers := []Event{newEvent(e.Config.Device, "forget", Tombstone{"raw-object", raw})}
		for _, m := range s.Messages {
			for _, a := range m.Attachments {
				if a.Blob != "" {
					markers = append(markers, newEvent(e.Config.Device, "forget", Tombstone{"object", a.Blob}))
				}
			}
		}
		if err = e.persist(markers); err != nil {
			e.mu.Unlock()
			return err
		}
		err = e.scrubForgotten()
		e.mu.Unlock()
		return err
	}
	if s.Updated == "" {
		s.Updated = info.ModTime().UTC().Format(time.RFC3339Nano)
	}
	if err = e.captureSession(s); err != nil {
		return err
	}
	// Only advance after durable raw archive and event commit. Partial tails are retried.
	if len(s.Issues) == 0 {
		_, err = e.DB.Exec("INSERT OR REPLACE INTO cursors VALUES(?,?,?,?)", path, info.Size(), info.ModTime().UnixNano(), raw)
	}
	return err
}
func queryObjects(tx *sql.Tx, query string, args ...any) ([]object, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []object
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err = rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := object{}
		for i, name := range cols {
			if b, ok := values[i].([]byte); ok {
				row[name] = string(b)
			} else {
				row[name] = values[i]
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
func (e *Engine) CaptureOpenCode(path string) (int, error) {
	e.assetMu.Lock()
	defer e.assetMu.Unlock()
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	_ = info
	u := url.URL{Scheme: "file", Path: path}
	db, err := sql.Open("sqlite", u.String()+"?mode=ro&_pragma=busy_timeout(3000)")
	if err != nil {
		return 0, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	sessions, err := queryObjects(tx, "SELECT * FROM session ORDER BY time_updated DESC")
	if err != nil {
		return 0, fmt.Errorf("OpenCode 来源格式暂不支持：%w", err)
	}
	for _, native := range sessions {
		id := str(native["id"])
		sid := key("opencode", id)
		if e.Forgotten("session", sid) {
			continue
		}
		stampKey := path + "#" + id
		var previous int64
		_ = e.DB.QueryRow("SELECT mtime FROM cursors WHERE path=?", stampKey).Scan(&previous)
		updated := num(native["time_updated"])
		var messagesUpdated, partsUpdated sql.NullInt64
		_ = tx.QueryRow("SELECT max(time_updated) FROM message WHERE session_id=?", id).Scan(&messagesUpdated)
		_ = tx.QueryRow("SELECT max(time_updated) FROM part WHERE session_id=?", id).Scan(&partsUpdated)
		updated = max(updated, messagesUpdated.Int64, partsUpdated.Int64)
		if previous == updated && previous != 0 {
			continue
		}
		messages, er := queryObjects(tx, "SELECT * FROM message WHERE session_id=? ORDER BY time_created,id", id)
		if er != nil {
			return len(sessions), er
		}
		parts, er := queryObjects(tx, "SELECT * FROM part WHERE session_id=? ORDER BY time_created,id", id)
		if er != nil {
			return len(sessions), er
		}
		raw, er := e.ArchiveRaw(bytes.NewReader(encoded(object{"session": native, "message": messages, "part": parts})), "opencode-native-row-snapshot")
		if er != nil {
			return len(sessions), er
		}
		partMap := map[string][]object{}
		for _, p := range parts {
			partMap[str(p["message_id"])] = append(partMap[str(p["message_id"])], p)
		}
		s := Session{ID: sid, NativeID: id, Agent: "opencode", Title: str(native["title"]), Created: stamp(native["time_created"]), Updated: stamp(native["time_updated"]), Raw: []string{raw}, Coverage: "一致性只读事务取得的会话原生记录；附件单独标明状态"}
		if parent := str(native["parent_id"]); parent != "" {
			s.Parent = key("opencode", parent)
		}
		for index, row := range messages {
			var data object
			if json.Unmarshal([]byte(str(row["data"])), &data) != nil {
				s.Issues = append(s.Issues, "消息数据无法解析："+str(row["id"]))
				continue
			}
			role := str(data["role"])
			if role != "user" && role != "assistant" {
				continue
			}
			var text []string
			var attachments []Attachment
			for _, p := range partMap[str(row["id"])] {
				var part object
				if json.Unmarshal([]byte(str(p["data"])), &part) != nil {
					continue
				}
				switch str(part["type"]) {
				case "text":
					if part["synthetic"] != true {
						text = append(text, str(part["text"]))
					}
				case "file":
					attachments = append(attachments, e.attachment(part))
				case "tool":
					state := obj(part["state"])
					if output := str(state["output"]); output != "" && !strings.HasPrefix(str(part["tool"]), "memvault") {
						text = append(text, "工具 "+str(part["tool"])+"：\n"+output)
					}
				}
			}
			s.Messages = append(s.Messages, Message{ID: str(row["id"]), Role: role, Text: strings.Join(text, "\n"), Time: stamp(row["time_created"]), Order: int64(index), Revision: num(row["time_updated"]), Complete: role == "user" || num(obj(data["time"])["completed"]) > 0, Attachments: attachments})
		}
		if er = e.captureSession(s); er != nil {
			return len(sessions), er
		}
		if _, er = e.DB.Exec("INSERT OR REPLACE INTO cursors VALUES(?,?,?,?)", stampKey, 0, updated, raw); er != nil {
			return len(sessions), er
		}
	}
	return len(sessions), nil
}
func (e *Engine) Scan(ctx context.Context) error {
	if !e.scanMu.TryLock() {
		return nil
	}
	defer e.scanMu.Unlock()
	if _, err := os.Stat(filepath.Join(e.Config.Root, "memvault.json")); err != nil {
		return err
	}
	e.assetMu.Lock()
	e.mu.Lock()
	err := e.Replay()
	if err == nil {
		var count int
		_ = e.DB.QueryRow("SELECT count(*) FROM tombstones").Scan(&count)
		if count > 0 {
			err = e.scrubForgotten()
		}
	}
	e.mu.Unlock()
	e.assetMu.Unlock()
	if err != nil {
		return err
	}
	for _, source := range e.configSnapshot().Sources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		status := SourceStatus{Agent: source.Agent, State: "ready", LastScan: utc()}
		if !source.Enabled {
			status.State = "paused"
		} else if source.Agent == "opencode" {
			status.Sessions, err = e.CaptureOpenCode(source.Path)
			if err != nil {
				status.State = "unavailable"
				status.Error = err.Error()
			}
		} else {
			var files []string
			err = filepath.WalkDir(source.Path, func(p string, d fs.DirEntry, er error) error {
				if er != nil {
					return er
				}
				if d.IsDir() {
					if source.Agent == "codex" && filepath.Dir(p) == source.Path && d.Name() != "sessions" && d.Name() != "archived_sessions" {
						return filepath.SkipDir
					}
					return nil
				}
				if (strings.HasSuffix(p, ".jsonl") || strings.HasSuffix(p, ".jsonl.zst")) && (source.Agent != "codex" || filepath.Dir(p) != source.Path || strings.HasPrefix(d.Name(), "rollout-")) {
					files = append(files, p)
				}
				return nil
			})
			if err != nil {
				status.State = "unavailable"
				status.Error = err.Error()
			} else {
				sort.Strings(files)
				status.State = "importing"
				e.statusMu.Lock()
				e.Statuses[source.Agent] = status
				e.statusMu.Unlock()
				e.notify()
				for _, p := range files {
					current := e.configSnapshot()
					enabled := false
					for _, s := range current.Sources {
						if s.Agent == source.Agent && s.Enabled && s.Path == source.Path {
							enabled = true
						}
					}
					if !enabled {
						status.State = "paused"
						break
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if er := e.CaptureFile(source.Agent, p, false); er != nil {
						status.State = "attention"
						status.Error = er.Error()
					}
					status.Sessions++
					status.LastScan = utc()
					e.statusMu.Lock()
					e.Statuses[source.Agent] = status
					e.statusMu.Unlock()
					e.notify()
				}
				if status.State == "importing" {
					status.State = "ready"
				}
			}
		}
		e.statusMu.Lock()
		e.Statuses[source.Agent] = status
		e.statusMu.Unlock()
		e.notify()
	}
	if err = e.WatchMemoryEdits(); err != nil {
		e.reportError(err)
	}
	return nil
}
func (e *Engine) Background(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		defer watcher.Close()
		for _, s := range e.configSnapshot().Sources {
			if s.Enabled {
				p := s.Path
				if info, er := os.Stat(p); er == nil && !info.IsDir() {
					p = filepath.Dir(p)
					_ = watcher.Add(p)
					continue
				}
				_ = filepath.WalkDir(p, func(path string, d fs.DirEntry, er error) error {
					if er == nil && d.IsDir() {
						if s.Agent == "codex" && filepath.Dir(path) == p && d.Name() != "sessions" && d.Name() != "archived_sessions" {
							return filepath.SkipDir
						}
						_ = watcher.Add(path)
					}
					return nil
				})
			}
		}
		_ = watcher.Add(filepath.Join(e.Config.Root, "journal"))
		_ = watcher.Add(filepath.Join(e.Config.Root, "memories"))
	}
	go func() {
		if er := e.ProtectedScan(ctx); er != nil {
			e.reportError(er)
		}
	}()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	var events <-chan fsnotify.Event
	var errors <-chan error
	if watcher != nil {
		events = watcher.Events
		errors = watcher.Errors
	}
	var debounce <-chan time.Time
	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if ev.Op&fsnotify.Create != 0 {
				if info, er := os.Stat(ev.Name); er == nil && info.IsDir() {
					_ = watcher.Add(ev.Name)
				}
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(700 * time.Millisecond)
			debounce = timer.C
		case er, ok := <-errors:
			if !ok {
				errors = nil
				continue
			}
			e.reportError(er)
		case <-debounce:
			debounce = nil
			go func() {
				if er := e.ProtectedScan(ctx); er != nil {
					e.reportError(er)
				}
			}()
		case <-ticker.C:
			go func() {
				if er := e.ProtectedScan(ctx); er != nil {
					e.reportError(er)
				}
			}()
		}
	}
}
func (e *Engine) reportError(err error) {
	if err == nil {
		return
	}
	e.statusMu.Lock()
	// Keep each distinct problem once, most recent last.
	next := make([]string, 0, len(e.ScanErrors)+1)
	for _, s := range e.ScanErrors {
		if s != err.Error() {
			next = append(next, s)
		}
	}
	next = append(next, err.Error())
	if len(next) > 30 {
		next = next[len(next)-30:]
	}
	e.ScanErrors = next
	e.statusMu.Unlock()
	e.notify()
}
func parseInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func sourceRevision(stamp string, line int64) int64 {
	if t, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
		return t.UnixNano()
	}
	return line
}
