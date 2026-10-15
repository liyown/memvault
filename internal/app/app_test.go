package app

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	t.Setenv("MEMVAULT_HOME", t.TempDir())
	t.Setenv("MEMVAULT_AGENT_HOME", t.TempDir())
	root := t.TempDir()
	must(t, Initialize(root))
	e, err := OpenEngine(Config{Root: root, Device: "test-device", Token: "test-token", ContextBudget: 16000})
	must(t, err)
	t.Cleanup(func() { e.Close() })
	return e
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T, agent, id string, texts ...string) string {
	t.Helper()
	var b bytes.Buffer
	if agent == "codex" {
		b.Write(encoded(object{"type": "session_meta", "payload": object{"id": id, "timestamp": "2026-10-15T10:00:00Z"}}))
		b.WriteByte('\n')
	}
	for i, text := range texts {
		row := object{"type": "user", "sessionId": id, "uuid": string(rune('a' + i)), "timestamp": "2026-10-15T10:00:00Z", "message": object{"role": "user", "content": text}}
		if agent == "codex" {
			row = object{"type": "response_item", "timestamp": "2026-10-15T10:00:00Z", "payload": object{"type": "message", "role": "user", "content": []object{{"type": "input_text", "text": text}}}}
		}
		b.Write(encoded(row))
		b.WriteByte('\n')
	}
	p := filepath.Join(t.TempDir(), id+".jsonl")
	must(t, os.WriteFile(p, b.Bytes(), 0600))
	return p
}
func TestArchiveIndependentDedupAndRebuild(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			e := testEngine(t)
			p := fixture(t, agent, "original", "中文记忆 portable archive", "中文记忆 portable archive")
			must(t, e.CaptureFile(agent, p, true))
			must(t, e.CaptureFile(agent, p, true))
			must(t, os.Remove(p))
			id := key(agent, "original")
			s, err := e.Session(id, 0, 100)
			must(t, err)
			if len(s.Messages) != 2 {
				t.Fatalf("identical distinct messages lost: %d", len(s.Messages))
			}
			raw, err := e.BlobRead(s.Raw[0])
			must(t, err)
			var manifest RawManifest
			must(t, json.Unmarshal(raw, &manifest))
			if len(manifest.Chunks) == 0 {
				t.Fatal("missing raw archive")
			}
			_, err = e.BlobRead(manifest.Chunks[0])
			must(t, err)
			hits, err := e.Search(SearchQuery{Query: "中文记忆", Limit: 30})
			must(t, err)
			if len(hits) != 2 {
				t.Fatalf("search = %d", len(hits))
			}
			must(t, e.Rebuild())
			s, err = e.Session(id, 0, 100)
			must(t, err)
			if len(s.Messages) != 2 {
				t.Fatal("rebuild lost messages")
			}
			body, err := e.ReadableSession(id)
			must(t, err)
			if !strings.Contains(body, "portable archive") {
				t.Fatal("readable missing original")
			}
		})
	}
}
func TestPartialTailAndMessageEdit(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "partial", "first")
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	must(t, err)
	_, err = f.WriteString(`{"type":"assistant","sessionId":"partial","uuid":"reply","message":{"role":"assistant","content":"`)
	must(t, err)
	f.Close()
	must(t, e.CaptureFile("claude", p, true))
	s, err := e.Session(key("claude", "partial"), 0, 100)
	must(t, err)
	if len(s.Messages) != 1 || len(s.Issues) == 0 {
		t.Fatal("partial tail not represented")
	}
	f, err = os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	must(t, err)
	_, err = f.WriteString("complete\"}}\n")
	must(t, err)
	f.Close()
	must(t, e.CaptureFile("claude", p, false))
	s, err = e.Session(s.ID, 0, 100)
	must(t, err)
	if len(s.Messages) != 2 || len(s.Issues) != 0 {
		t.Fatal("tail not recovered")
	}
	edit := s.Messages[1]
	edit.Text = "edited"
	edit.Revision++
	must(t, e.persist([]Event{newEvent(e.Config.Device, "message", Change{Session: s.ID, Message: &edit})}))
	must(t, e.Rebuild())
	s, err = e.Session(s.ID, 0, 100)
	must(t, err)
	if s.Messages[1].Text != "edited" {
		t.Fatal("version update lost")
	}
}
func TestMemoryConflictsCoverageAndMarkdown(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "codex", "summary", "用户明确要求本地保存")
	must(t, e.CaptureFile("codex", p, true))
	sid := key("codex", "summary")
	c, err := e.Context(sid, "")
	must(t, err)
	pending := c["pending_messages"].([]map[string]any)
	version := pending[0]["version"].(string)
	m, err := e.Commit(Memory{Title: "会话摘要", Kind: "summary", Session: sid, Body: "本地保存", Covered: []string{version}, Evidence: []Evidence{{Session: sid, Message: "line:2"}}})
	must(t, err)
	c, err = e.Context(sid, "")
	must(t, err)
	if len(c["pending_messages"].([]map[string]any)) != 0 {
		t.Fatal("coverage ignored")
	}
	a := m
	a.Body = "分支 A"
	a.Parents = []string{m.Revision}
	_, err = e.Commit(a)
	must(t, err)
	b := m
	b.Body = "分支 B"
	b.Parents = []string{m.Revision}
	_, err = e.Commit(b)
	must(t, err)
	heads, err := e.Memories(m.ID, false)
	must(t, err)
	if len(heads) != 2 {
		t.Fatalf("lost concurrent branch: %d", len(heads))
	}
	a.Body = "合并版本"
	a.Parents = []string{heads[0].Revision, heads[1].Revision}
	merged, err := e.Commit(a)
	must(t, err)
	path := filepath.Join(e.Config.Root, "memories", m.ID+".md")
	raw, err := os.ReadFile(path)
	must(t, err)
	must(t, os.WriteFile(path, bytes.Replace(raw, []byte("\n\n合并版本\n"), []byte("\n\n人工编辑\n"), 1), 0600))
	must(t, e.WatchMemoryEdits())
	heads, err = e.Memories(m.ID, false)
	must(t, err)
	if len(heads) != 1 || heads[0].Body != "人工编辑" || heads[0].Parents[0] != merged.Revision {
		t.Fatal("Markdown version not captured")
	}
}
func TestExportImportForgetAndOldReplica(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "secret", "private-fact-needle")
	must(t, e.CaptureFile("claude", p, true))
	id := key("claude", "secret")
	var b bytes.Buffer
	must(t, e.Export(&b))
	export := filepath.Join(t.TempDir(), "archive.zip")
	must(t, os.WriteFile(export, b.Bytes(), 0600))
	otherRoot := t.TempDir()
	must(t, Initialize(otherRoot))
	c := e.Config
	c.Root = otherRoot
	c.Device = "device-b"
	other, err := OpenEngine(c)
	must(t, err)
	defer other.Close()
	must(t, other.Import(export))
	s, err := other.Session(id, 0, 100)
	must(t, err)
	if len(s.Messages) != 1 {
		t.Fatal("import lost message")
	}
	must(t, other.Forget("session", id, true))
	must(t, other.Import(export))
	must(t, other.Rebuild())
	if _, err = other.Session(id, 0, 10); err == nil {
		t.Fatal("deleted data resurrected")
	}
	must(t, filepath.WalkDir(otherRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if bytes.Contains(raw, []byte("private-fact-needle")) {
				t.Errorf("forgotten body persisted: %s", p)
			}
		}
		return nil
	}))
}
func TestOpenCodeSnapshotAndAttachments(t *testing.T) {
	e := testEngine(t)
	p := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", p)
	must(t, err)
	_, err = db.Exec(`CREATE TABLE session(id TEXT,title TEXT,time_created INTEGER,time_updated INTEGER,parent_id TEXT);CREATE TABLE message(id TEXT,session_id TEXT,time_created INTEGER,time_updated INTEGER,data TEXT);CREATE TABLE part(id TEXT,message_id TEXT,session_id TEXT,time_created INTEGER,time_updated INTEGER,data TEXT);INSERT INTO session VALUES('s','来源会话',1000,2000,NULL);INSERT INTO message VALUES('m','s',1000,2000,'{"role":"user"}');INSERT INTO part VALUES('p','m','s',1000,2000,'{"type":"text","text":"portable SQLite 对话"}');INSERT INTO part VALUES('image','m','s',1000,2000,'{"type":"file","filename":"pixel.png","url":"data:image/png;base64,aGVsbG8="}');`)
	must(t, err)
	db.Close()
	n, err := e.CaptureOpenCode(p)
	must(t, err)
	if n != 1 {
		t.Fatal(n)
	}
	must(t, os.Remove(p))
	s, err := e.Session(key("opencode", "s"), 0, 100)
	must(t, err)
	if len(s.Messages) != 1 || len(s.Messages[0].Attachments) != 1 {
		t.Fatalf("parsed %#v", s)
	}
	a := s.Messages[0].Attachments[0]
	raw, err := e.BlobRead(a.Blob)
	must(t, err)
	if string(raw) != "hello" {
		t.Fatal("attachment unavailable")
	}
}
func TestHTTPIsolationAndSafeRendering(t *testing.T) {
	e := testEngine(t)
	server := NewServer(e, "127.0.0.1:27941")
	h := server.Handler()
	for _, tc := range []struct {
		host, origin, token string
		want                int
	}{{"127.0.0.1:27941", "", "", 401}, {"evil.example", "", "test-token", 403}, {"127.0.0.1:27941", "https://evil.example", "test-token", 403}, {"127.0.0.1:27941", "", "test-token", 200}} {
		r := httptest.NewRequest("GET", "http://"+tc.host+"/api/v1/status", nil)
		r.Host = tc.host
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		r.Header.Set("Authorization", "Bearer "+tc.token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("status %d want %d", w.Code, tc.want)
		}
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:27941/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'self'") {
		t.Fatal("missing CSP")
	}
}
func TestConfigMergeIdempotentAndRemoval(t *testing.T) {
	e := testEngine(t)
	t.Setenv("CODEX_HOME", filepath.Join(home(), ".codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home(), ".claude"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home(), ".config"))
	e.Config.Binary = "/test path/memvault"
	path := filepath.Join(home(), ".claude", "settings.json")
	must(t, atomicWrite(path, []byte(`{"env":{"KEEP":"value"},"hooks":{"Stop":[{"hooks":[{"type":"command","command":"existing-hook"}]}]}}`), 0600))
	a, err := Configure(e.Config, false)
	must(t, err)
	for _, x := range a {
		if x.Error != "" {
			t.Fatal(x)
		}
	}
	before, err := os.ReadFile(path)
	must(t, err)
	_, err = Configure(e.Config, false)
	must(t, err)
	after, err := os.ReadFile(path)
	must(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("not idempotent")
	}
	_, err = Configure(e.Config, true)
	must(t, err)
	after, err = os.ReadFile(path)
	must(t, err)
	if !bytes.Contains(after, []byte("existing-hook")) || !bytes.Contains(after, []byte("KEEP")) || bytes.Contains(after, []byte("--integration memvault-v1")) {
		t.Fatal("unrelated configuration changed")
	}
}
func TestOfflineRecoveryAndLease(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "offline", "queue restored")
	e.Config.Sources = []Source{{Agent: "claude", Path: filepath.Dir(p), Enabled: true}}
	old := e.Config.Root + "-away"
	must(t, os.Rename(e.Config.Root, old))
	if e.ProtectedScan(context.Background()) == nil {
		t.Fatal("unavailable root reported healthy")
	}
	must(t, os.Rename(old, e.Config.Root))
	must(t, e.ProtectedScan(context.Background()))
	_, err := e.Session(key("claude", "offline"), 0, 10)
	must(t, err)
	_, err = e.Claim("task", "agent-a")
	must(t, err)
	if _, err = e.Claim("task", "agent-b"); err == nil {
		t.Fatal("competing claim succeeded")
	}
}
func TestZipTraversalRejectedAndMCP(t *testing.T) {
	e := testEngine(t)
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, err := z.Create("../escape")
	must(t, err)
	_, err = io.WriteString(f, "bad")
	must(t, err)
	must(t, z.Close())
	path := filepath.Join(t.TempDir(), "bad.zip")
	must(t, os.WriteFile(path, b.Bytes(), 0600))
	if e.Import(path) == nil {
		t.Fatal("unsafe import accepted")
	}
	in := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	var out bytes.Buffer
	must(t, MCP(e.Config, in, &out))
	for _, tool := range toolsList() {
		if tool.InputSchema["required"] == nil {
			t.Fatal("JSON Schema required must be an array")
		}
	}
	if len(toolsList()) != 6 || !strings.Contains(out.String(), "memvault_session_read") {
		t.Fatal("MCP contract incomplete")
	}
}

func TestDurableJournalRecoversBeforeIndexCommit(t *testing.T) {
	e := testEngine(t)
	s := Session{ID: key("codex", "crash"), NativeID: "crash", Agent: "codex", Title: "中断恢复", Updated: utc()}
	m := Message{ID: "m", Role: "user", Text: "已写原档，索引尚未提交", Order: 1, Complete: true}
	events := []Event{newEvent(e.Config.Device, "session", s), newEvent(e.Config.Device, "message", Change{Session: s.ID, Message: &m})}
	var b bytes.Buffer
	for _, ev := range events {
		b.Write(encoded(ev))
		b.WriteByte('\n')
	}
	must(t, atomicWrite(filepath.Join(e.Config.Root, "journal", "other-device", hash(b.Bytes())+".jsonl"), b.Bytes(), 0600))
	// This is the on-disk state after a process dies between journal fsync and SQLite commit.
	recovered, err := OpenEngine(e.Config)
	must(t, err)
	defer recovered.Close()
	got, err := recovered.Session(s.ID, 0, 10)
	must(t, err)
	if len(got.Messages) != 1 || got.Messages[0].Text != m.Text {
		t.Fatal("durable event not recovered")
	}
	must(t, recovered.Replay())
	got, err = recovered.Session(s.ID, 0, 10)
	must(t, err)
	if len(got.Messages) != 1 {
		t.Fatal("replay duplicated message")
	}
}

func TestMissingObjectAndCorruptSegmentReported(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "verify", "原文完整性检查")
	must(t, e.CaptureFile("claude", p, true))
	s, err := e.Session(key("claude", "verify"), 0, 10)
	must(t, err)
	raw, err := e.BlobRead(s.Raw[0])
	must(t, err)
	var manifest RawManifest
	must(t, json.Unmarshal(raw, &manifest))
	id := manifest.Chunks[0]
	must(t, os.Remove(filepath.Join(e.Config.Root, "objects", id[:2], id)))
	if len(e.Verify()) == 0 {
		t.Fatal("missing raw object silently accepted")
	}
	bad := strings.Repeat("a", 64) + ".jsonl"
	must(t, atomicWrite(filepath.Join(e.Config.Root, "journal", "other", bad), []byte("corrupt"), 0600))
	must(t, e.Replay())
	if len(e.ScanErrors) == 0 {
		t.Fatal("corruption not reported")
	}
}

func TestContinuationAndNativeSubsessions(t *testing.T) {
	e := testEngine(t)
	parent := key("codex", "parent")
	p := fixture(t, "codex", "child", "[memvault-continue:"+parent+"] 请继续")
	must(t, e.CaptureFile("codex", p, true))
	links, err := e.Relations(key("codex", "child"))
	must(t, err)
	if len(links) != 1 || links[0].Kind != "continuation" {
		t.Fatal("continuation identity lost")
	}
	data := []byte("{\"type\":\"user\",\"sessionId\":\"parent\",\"uuid\":\"m\",\"message\":{\"role\":\"user\",\"content\":\"subagent task\"}}\n")
	child, err := e.ParseTranscript("claude", data, "agent-example")
	must(t, err)
	if child.ID == key("claude", "parent") || child.Parent != key("claude", "parent") {
		t.Fatal("subsession mixed into parent")
	}
}

func TestFutureSchemaProtected(t *testing.T) {
	root := t.TempDir()
	must(t, os.WriteFile(filepath.Join(root, "memvault.json"), []byte(`{"schema":999,"id":"future"}`), 0600))
	if Initialize(root) == nil {
		t.Fatal("unknown format was modified")
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("unknown archive not read-only")
	}
}

func TestRestartWhileRootUnavailable(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "restart-offline", "restart queued")
	e.Config.Sources = []Source{{Agent: "claude", Path: filepath.Dir(p), Enabled: true}}
	away := e.Config.Root + "-away"
	must(t, os.Rename(e.Config.Root, away))
	restarted, err := OpenEngine(e.Config)
	must(t, err)
	defer restarted.Close()
	if restarted.ProtectedScan(context.Background()) == nil {
		t.Fatal("missing storage reported healthy")
	}
	must(t, os.Rename(away, e.Config.Root))
	must(t, restarted.ProtectedScan(context.Background()))
	_, err = restarted.Session(key("claude", "restart-offline"), 0, 10)
	must(t, err)
}

func TestChronologicalSearchSurvivesRebuild(t *testing.T) {
	e := testEngine(t)
	s := Session{ID: key("claude", "sort"), Agent: "claude", NativeID: "sort", Title: "搜索排序", Updated: utc()}
	s.Messages = []Message{{ID: "old", Role: "user", Text: "共同关键词 archive", Time: "2026-01-01T00:00:00Z", Order: 1, Complete: true}, {ID: "new", Role: "user", Text: "共同关键词 archive", Time: "2026-10-01T00:00:00Z", Order: 2, Complete: true}}
	must(t, e.captureSession(s))
	hits, err := e.Search(SearchQuery{Query: "共同关键词", Limit: 30})
	must(t, err)
	if len(hits) != 2 || hits[0].Message != "new" {
		t.Fatal("recent order incorrect")
	}
	s.Messages[0].Text = "共同关键词 archive 已修订"
	s.Messages[0].Revision = 1
	must(t, e.captureSession(s))
	must(t, e.Rebuild())
	hits, err = e.Search(SearchQuery{Query: "共同关键词", Limit: 30})
	must(t, err)
	if len(hits) != 2 || hits[0].Message != "new" {
		t.Fatal("index update duplicated or reordered messages")
	}
	var refs int
	must(t, e.DB.QueryRow("SELECT count(*) FROM search_refs").Scan(&refs))
	if refs != 2 {
		t.Fatal("identity map mismatch", refs)
	}
}

func TestRotationRetainsEditedNativeMessage(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "rotation", "old body")
	must(t, e.CaptureFile("claude", p, true))
	row := object{"type": "user", "sessionId": "rotation", "uuid": "a", "timestamp": "2026-10-16T10:00:00Z", "message": object{"role": "user", "content": "edited after file rotation"}}
	must(t, os.WriteFile(p, append(encoded(row), '\n'), 0600))
	must(t, e.CaptureFile("claude", p, true))
	must(t, e.Rebuild())
	s, err := e.Session(key("claude", "rotation"), 0, 10)
	must(t, err)
	if len(s.Messages) != 1 || s.Messages[0].Text != "edited after file rotation" {
		t.Fatal("rotation lost native identity or update")
	}
}

func TestCorruptIndexRecreatedFromArchive(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "codex", "index-recovery", "原始资料不依赖索引")
	must(t, e.CaptureFile("codex", p, true))
	var manifest Manifest
	must(t, readJSON(filepath.Join(e.Config.Root, "memvault.json"), &manifest))
	must(t, e.Close())
	index := filepath.Join(stateDir(), manifest.ID, "index.sqlite")
	must(t, os.WriteFile(index, []byte("not a sqlite database"), 0600))
	recovered, err := OpenEngine(e.Config)
	must(t, err)
	defer recovered.Close()
	s, err := recovered.Session(key("codex", "index-recovery"), 0, 10)
	must(t, err)
	if len(s.Messages) != 1 {
		t.Fatal("corrupt cache prevented recovery")
	}
}

func TestForgetPreservesUnrelatedPartialSync(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "forget-partial", "forget this body")
	must(t, e.CaptureFile("claude", p, true))
	incoming, err := e.Blob([]byte("incoming object whose journal has not arrived"))
	must(t, err)
	must(t, e.Forget("session", key("claude", "forget-partial"), true))
	b, err := e.BlobRead(incoming)
	must(t, err)
	if !strings.Contains(string(b), "incoming object") {
		t.Fatal("partial sync object deleted as garbage")
	}
	// Replayed cleanup also preserves unowned arrivals, while known deleted
	// objects remain tracked in content-free tombstones for interrupted cleanup.
	must(t, e.Rebuild())
	must(t, e.scrubForgotten())
	_, err = e.BlobRead(incoming)
	must(t, err)
}

func TestJSONAttachmentIsNotAnArchiveManifest(t *testing.T) {
	e := testEngine(t)
	blob, err := e.Blob([]byte(`{"format":"custom-data","chunks":["not-an-archive-object"]}`))
	must(t, err)
	s := Session{ID: key("claude", "json"), NativeID: "json", Agent: "claude", Title: "JSON attachment", Updated: utc(), Messages: []Message{{ID: "m", Role: "user", Text: "保留 JSON 文件", Complete: true, Attachments: []Attachment{{Name: "custom.json", Blob: blob}}}}}
	must(t, e.captureSession(s))
	if issues := e.Verify(); len(issues) > 0 {
		t.Fatal("opaque JSON attachment misparsed", issues)
	}
}

func TestForgottenSourceChangesDoNotRetainNewPayload(t *testing.T) {
	e := testEngine(t)
	p := fixture(t, "claude", "forgotten-window", "original-private-needle")
	must(t, e.CaptureFile("claude", p, true))
	must(t, e.Forget("session", key("claude", "forgotten-window"), true))
	changed := fixture(t, "claude", "forgotten-window", "new-private-needle")
	b, err := os.ReadFile(changed)
	must(t, err)
	must(t, os.WriteFile(p, b, 0600))
	must(t, e.CaptureFile("claude", p, true))
	must(t, e.Rebuild())
	if _, err := e.Session(key("claude", "forgotten-window"), 0, 10); err == nil {
		t.Fatal("changed source resurrected forgotten session")
	}
	must(t, filepath.WalkDir(e.Config.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if bytes.Contains(body, []byte("private-needle")) {
			t.Errorf("forgotten source payload retained: %s", path)
		}
		return err
	}))
}

func TestContextKeepsIdentityBeforeArchive(t *testing.T) {
	e := testEngine(t)
	sid := key("claude", "not-yet-archived")
	value, err := e.Context(sid, "")
	must(t, err)
	if s := value["session"].(Session); s.ID != sid {
		t.Fatal("unarchived session lost its identity", s)
	}
}

func TestReportErrorDeduplicates(t *testing.T) {
	e := testEngine(t)
	for i := 0; i < 5; i++ {
		e.reportError(os.ErrNotExist)
		e.reportError(io.ErrUnexpectedEOF)
	}
	if len(e.ScanErrors) != 2 {
		t.Fatal("repeated errors not deduplicated", e.ScanErrors)
	}
}

func TestInboxDropsMissingTranscript(t *testing.T) {
	e := testEngine(t)
	s := NewServer(e, "127.0.0.1")
	file := filepath.Join(stateDir(), "inbox", "gone.json")
	must(t, atomicWrite(file, encoded(object{"agent": "claude", "transcript_path": filepath.Join(t.TempDir(), "deleted.jsonl")}), 0600))
	s.processInbox(context.Background())
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("envelope for deleted transcript retried forever")
	}
	if len(e.ScanErrors) != 0 {
		t.Fatal("deleted transcript reported as error", e.ScanErrors)
	}
}
