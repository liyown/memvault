package app

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	_ "modernc.org/sqlite"
)

type Engine struct {
	Config     Config
	DB         *sql.DB
	mu         sync.Mutex
	assetMu    sync.Mutex
	configMu   sync.RWMutex
	protectMu  sync.Mutex
	scanMu     sync.Mutex
	statusMu   sync.RWMutex
	Statuses   map[string]SourceStatus
	ScanErrors []string
	changed    chan struct{}
}
type Manifest struct {
	Schema  int    `json:"schema"`
	ID      string `json:"id"`
	Created string `json:"created"`
}

func Initialize(root string) error {
	root, e := filepath.Abs(root)
	if e != nil {
		return e
	}
	p := filepath.Join(root, "memvault.json")
	if _, e = os.Stat(p); errors.Is(e, os.ErrNotExist) {
		if e = atomicWrite(p, encoded(Manifest{Schema, randomID(), utc()}), 0600); e != nil {
			return e
		}
	} else if e != nil {
		return e
	}
	var manifest Manifest
	if err := readJSON(p, &manifest); err != nil {
		return err
	}
	if manifest.Schema != Schema {
		return fail("档案格式版本不受支持；未改写")
	}
	for _, d := range []string{"objects", "journal", "memories", "sessions", "quarantine"} {
		if e = os.MkdirAll(filepath.Join(root, d), 0700); e != nil {
			return e
		}
	}
	for path, body := range map[string]string{"README.md": vaultReadme, "AGENTS.md": agentProtocol, "FORMAT.md": formatDoc} {
		p := filepath.Join(root, path)
		if _, e = os.Stat(p); errors.Is(e, os.ErrNotExist) {
			if e = atomicWrite(p, []byte(body), 0600); e != nil {
				return e
			}
		}
	}
	return nil
}
func OpenEngine(c Config) (*Engine, error) {
	var m Manifest
	available := true
	if err := readJSON(filepath.Join(c.Root, "memvault.json"), &m); err != nil {
		var cached struct {
			Root     string   `json:"root"`
			Manifest Manifest `json:"manifest"`
		}
		if (!os.IsNotExist(err) && !os.IsPermission(err)) || readJSON(filepath.Join(configDir(), "vault-cache.json"), &cached) != nil || cached.Root != c.Root {
			return nil, err
		}
		m = cached.Manifest
		available = false
	}
	if available && c.Root != filepath.Join(stateDir(), "offline-vault") {
		if err := atomicWrite(filepath.Join(configDir(), "vault-cache.json"), encoded(object{"root": c.Root, "manifest": m}), 0600); err != nil {
			return nil, err
		}
	}
	if m.Schema != Schema {
		return nil, fail("档案格式版本 %d 不受支持；未修改数据", m.Schema)
	}
	dir := filepath.Join(stateDir(), m.ID)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", filepath.Join(dir, "index.sqlite"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	engine := &Engine{Config: c, DB: db, Statuses: map[string]SourceStatus{}, changed: make(chan struct{}, 1)}
	if _, e = db.Exec(schemaSQL); e != nil {
		db.Close()
		var coded interface{ Code() int }
		if !errors.As(e, &coded) || (coded.Code()&255 != 11 && coded.Code()&255 != 26) {
			return nil, e
		}
		indexPath := filepath.Join(dir, "index.sqlite")
		suffix := ".corrupt-" + randomID()
		for _, ext := range []string{"", "-wal", "-shm"} {
			if err := os.Rename(indexPath+ext, indexPath+suffix+ext); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
		}
		db, e = sql.Open("sqlite", indexPath)
		if e != nil {
			return nil, e
		}
		db.SetMaxOpenConns(1)
		engine.DB = db
		if _, e = db.Exec(schemaSQL); e != nil {
			db.Close()
			return nil, e
		}
	}
	var indexed, refs int
	_ = db.QueryRow("SELECT count(*) FROM search").Scan(&indexed)
	_ = db.QueryRow("SELECT count(*) FROM search_refs").Scan(&refs)
	if available && indexed > 0 && refs == 0 {
		_, e = db.Exec(`DELETE FROM segments;DELETE FROM applied;DELETE FROM sessions;DELETE FROM messages;DELETE FROM memories;DELETE FROM parents;DELETE FROM relations;DELETE FROM search;DELETE FROM cursors;`)
		if e != nil {
			db.Close()
			return nil, e
		}
	}
	if available {
		e = engine.Replay()
	}
	if e != nil {
		db.Close()
		return nil, e
	}
	return engine, nil
}
func (e *Engine) Close() error { return e.DB.Close() }

const schemaSQL = `
PRAGMA journal_mode=WAL;
PRAGMA synchronous=FULL;
PRAGMA secure_delete=ON;
PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS segments(path TEXT PRIMARY KEY,size INTEGER,mtime INTEGER);
CREATE TABLE IF NOT EXISTS search_refs(entity TEXT PRIMARY KEY,rowid INTEGER UNIQUE,session TEXT,memory TEXT);
CREATE INDEX IF NOT EXISTS search_refs_session ON search_refs(session);
CREATE INDEX IF NOT EXISTS search_refs_memory ON search_refs(memory);
CREATE TABLE IF NOT EXISTS applied(id TEXT PRIMARY KEY,segment TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(id TEXT PRIMARY KEY,agent TEXT,native_id TEXT,title TEXT,created TEXT,updated TEXT,data BLOB);
CREATE INDEX IF NOT EXISTS sessions_updated ON sessions(updated DESC);
CREATE TABLE IF NOT EXISTS messages(session TEXT,id TEXT,version TEXT,seq INTEGER,role TEXT,text TEXT,time TEXT,revision INTEGER,complete INTEGER,data BLOB,PRIMARY KEY(session,id));
CREATE INDEX IF NOT EXISTS messages_session_order ON messages(session,seq);
CREATE TABLE IF NOT EXISTS memories(id TEXT,revision TEXT,title TEXT,kind TEXT,session TEXT,body TEXT,updated TEXT,data BLOB,PRIMARY KEY(id,revision));
CREATE TABLE IF NOT EXISTS parents(memory TEXT,revision TEXT,parent TEXT,PRIMARY KEY(memory,revision,parent));
CREATE TABLE IF NOT EXISTS relations(origin TEXT,target TEXT,kind TEXT,PRIMARY KEY(origin,target,kind));
CREATE TABLE IF NOT EXISTS tombstones(kind TEXT,id TEXT,PRIMARY KEY(kind,id));
CREATE TABLE IF NOT EXISTS leases(id TEXT PRIMARY KEY,owner TEXT,until_time TEXT);
CREATE TABLE IF NOT EXISTS cursors(path TEXT PRIMARY KEY,size INTEGER,mtime INTEGER,digest TEXT);
CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(tokens,text UNINDEXED,session UNINDEXED,memory UNINDEXED,message UNINDEXED,title UNINDEXED,agent UNINDEXED,role UNINDEXED,time UNINDEXED);
INSERT INTO search(search,rank) VALUES('secure-delete',1);
`

func (e *Engine) notify() {
	select {
	case e.changed <- struct{}{}:
	default:
	}
}
func (e *Engine) Blob(b []byte) (string, error) {
	id := hash(b)
	p := filepath.Join(e.Config.Root, "objects", id[:2], id)
	if _, err := os.Stat(p); err == nil {
		return id, nil
	}
	return id, atomicWrite(p, b, 0600)
}
func (e *Engine) BlobRead(id string) ([]byte, error) {
	if len(id) != 64 || checkID(id) != nil {
		return nil, fail("无效内容编号")
	}
	p := filepath.Join(e.Config.Root, "objects", id[:2], id)
	b, err := os.ReadFile(p)
	if err == nil && hash(b) != id {
		return nil, fail("档案校验失败：%s", id)
	}
	return b, err
}
func newEvent(device, kind string, v any) Event {
	p := encoded(v)
	return Event{Schema: Schema, ID: hash(append([]byte(kind+":"), p...)), Device: device, Kind: kind, Time: utc(), Payload: p}
}
func (e *Engine) persist(events []Event) error {
	var pending []Event
	for _, ev := range events {
		var n int
		if err := e.DB.QueryRow("SELECT count(*) FROM applied WHERE id=?", ev.ID).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			pending = append(pending, ev)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	// Bounded sealed batches. No shared append file, and the journal is durable before indexing.
	for start := 0; start < len(pending); start += 256 {
		end := start + 256
		if end > len(pending) {
			end = len(pending)
		}
		var b strings.Builder
		for _, ev := range pending[start:end] {
			b.Write(encoded(ev))
			b.WriteByte('\n')
		}
		data := []byte(b.String())
		name := filepath.Join("journal", e.Config.Device, hash(data)+".jsonl")
		if err := atomicWrite(filepath.Join(e.Config.Root, name), data, 0600); err != nil {
			return err
		}
		if err := e.applyBatch(pending[start:end], name); err != nil {
			return err
		}
	}
	e.notify()
	return nil
}
func (e *Engine) applyBatch(events []Event, segment string) error {
	tx, err := e.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, ev := range events {
		if ev.Schema != Schema {
			return fail("不支持的记录版本：%d", ev.Schema)
		}
		if hash(append([]byte(ev.Kind+":"), ev.Payload...)) != ev.ID {
			return fail("档案记录校验失败：%s", ev.ID)
		}
		r, err := tx.Exec("INSERT OR IGNORE INTO applied VALUES(?,?)", ev.ID, segment)
		if err != nil {
			return err
		}
		n, _ := r.RowsAffected()
		if n == 0 {
			continue
		}
		if err = e.apply(tx, ev); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func dead(tx *sql.Tx, kind, id string) bool {
	var n int
	_ = tx.QueryRow("SELECT count(*) FROM tombstones WHERE kind=? AND id=?", kind, id).Scan(&n)
	return n > 0
}
func (e *Engine) Forgotten(kind, id string) bool {
	var n int
	_ = e.DB.QueryRow("SELECT count(*) FROM tombstones WHERE kind=? AND id=?", kind, id).Scan(&n)
	return n > 0
}
func (e *Engine) apply(tx *sql.Tx, ev Event) error {
	switch ev.Kind {
	case "session":
		var s Session
		if err := json.Unmarshal(ev.Payload, &s); err != nil {
			return err
		}
		if dead(tx, "session", s.ID) {
			return nil
		}
		_, err := tx.Exec(`INSERT INTO sessions VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,updated=excluded.updated,data=excluded.data WHERE excluded.updated>=sessions.updated`, s.ID, s.Agent, s.NativeID, s.Title, s.Created, s.Updated, ev.Payload)
		return err
	case "message":
		var c Change
		if err := json.Unmarshal(ev.Payload, &c); err != nil {
			return err
		}
		m := c.Message
		if m == nil {
			return fail("消息记录缺失")
		}
		if dead(tx, "session", c.Session) || dead(tx, "message", c.Session+":"+m.ID) {
			return nil
		}
		var old Message
		var oldData []byte
		if err := tx.QueryRow("SELECT data FROM messages WHERE session=? AND id=?", c.Session, m.ID).Scan(&oldData); err == nil {
			_ = json.Unmarshal(oldData, &old)
			if old.Revision > m.Revision || (old.Revision == m.Revision && old.Complete && !m.Complete) {
				return nil
			}
		}
		_, err := tx.Exec(`INSERT OR REPLACE INTO messages VALUES(?,?,?,?,?,?,?,?,?,?)`, c.Session, m.ID, ev.ID, m.Order, m.Role, m.Text, m.Time, m.Revision, m.Complete, encoded(m))
		if err != nil {
			return err
		}
		err = searchDelete(tx, "entity", "message:"+c.Session+":"+m.ID)
		if err != nil {
			return err
		}
		var title, agent string
		_ = tx.QueryRow("SELECT title,agent FROM sessions WHERE id=?", c.Session).Scan(&title, &agent)
		err = searchInsert(tx, "message:"+c.Session+":"+m.ID, first(m.Time, ev.Time), []any{terms(m.Text + " " + title), m.Text, c.Session, "", m.ID, title, agent, m.Role, m.Time})
		return err
	case "memory":
		var m Memory
		if err := json.Unmarshal(ev.Payload, &m); err != nil {
			return err
		}
		if dead(tx, "memory", m.ID) || (m.Session != "" && dead(tx, "session", m.Session)) {
			return nil
		}
		m.Revision = ev.ID
		_, err := tx.Exec("INSERT OR IGNORE INTO memories VALUES(?,?,?,?,?,?,?,?)", m.ID, ev.ID, m.Title, m.Kind, m.Session, m.Body, m.Updated, encoded(m))
		if err != nil {
			return err
		}
		for _, p := range m.Parents {
			if _, err = tx.Exec("INSERT OR IGNORE INTO parents VALUES(?,?,?)", m.ID, ev.ID, p); err != nil {
				return err
			}
		}
		err = searchDelete(tx, "memory", m.ID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(`SELECT data FROM memories m WHERE id=? AND NOT EXISTS(SELECT 1 FROM parents p WHERE p.memory=m.id AND p.parent=m.revision)`, m.ID)
		if err != nil {
			return err
		}
		var heads []Memory
		for rows.Next() {
			var b []byte
			_ = rows.Scan(&b)
			var x Memory
			_ = json.Unmarshal(b, &x)
			heads = append(heads, x)
		}
		rows.Close()
		for _, x := range heads {
			if err = searchInsert(tx, "memory:"+x.ID+":"+x.Revision, x.Updated, []any{terms(x.Title + " " + x.Body), x.Body, x.Session, x.ID, "", x.Title, "", "memory", x.Updated}); err != nil {
				return err
			}
		}
		return nil
	case "relation":
		var r Relation
		if err := json.Unmarshal(ev.Payload, &r); err != nil {
			return err
		}
		if dead(tx, "session", r.From) || dead(tx, "session", r.To) {
			return nil
		}
		_, err := tx.Exec("INSERT OR IGNORE INTO relations VALUES(?,?,?)", r.From, r.To, r.Kind)
		return err
	case "forget":
		var t Tombstone
		if err := json.Unmarshal(ev.Payload, &t); err != nil {
			return err
		}
		_, err := tx.Exec("INSERT OR IGNORE INTO tombstones VALUES(?,?)", t.Kind, t.ID)
		if err != nil {
			return err
		}
		if t.Kind == "object" || t.Kind == "raw-object" {
			return nil
		}
		if t.Kind == "session" {
			_, err = tx.Exec("DELETE FROM messages WHERE session=?", t.ID)
			if err != nil {
				return err
			}
			_, err = tx.Exec("DELETE FROM sessions WHERE id=?", t.ID)
			if err != nil {
				return err
			}
			err = searchDelete(tx, "session", t.ID)
			if err != nil {
				return err
			}
			_, err = tx.Exec("DELETE FROM memories WHERE session=?", t.ID)
			if err != nil {
				return err
			}
			_, err = tx.Exec("DELETE FROM relations WHERE origin=? OR target=?", t.ID, t.ID)
			return err
		} else if t.Kind == "memory" {
			err = searchDelete(tx, "memory", t.ID)
			if err != nil {
				return err
			}
			_, err = tx.Exec("DELETE FROM memories WHERE id=?", t.ID)
			return err
		}
		return fail("未知遗忘类型")
	default:
		return fail("未知事件类型 %s；已保留原始档案", ev.Kind)
	}
}
func (e *Engine) Replay() error {
	var segments []string
	err := filepath.WalkDir(filepath.Join(e.Config.Root, "journal"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".jsonl") {
			segments = append(segments, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// First collect deletion markers so arrival or clock order never resurrects forgotten data.
	var batches [][]Event
	var names []string
	for _, p := range segments {
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		var size, mtime int64
		_ = e.DB.QueryRow("SELECT size,mtime FROM segments WHERE path=?", p).Scan(&size, &mtime)
		if size == info.Size() && mtime == info.ModTime().UnixNano() {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		base := strings.TrimSuffix(filepath.Base(p), ".jsonl")
		if len(base) == 64 && hash(b) != base {
			e.reportError(fail("校验失败：%s", filepath.Base(p)))
			continue
		}
		var events []Event
		scanner := bufio.NewScanner(strings.NewReader(string(b)))
		scanner.Buffer(make([]byte, 65536), 64<<20)
		bad := false
		for scanner.Scan() {
			var ev Event
			if err = json.Unmarshal(scanner.Bytes(), &ev); err != nil {
				bad = true
				break
			}
			if ev.Schema > Schema {
				return fail("发现不支持的记录版本 %d，保留原文且停止改写", ev.Schema)
			}
			events = append(events, ev)
		}
		if scanner.Err() != nil {
			bad = true
		}
		if bad {
			e.reportError(fail("记录无法读取：%s", filepath.Base(p)))
			continue
		}
		for _, ev := range events {
			if ev.Kind == "forget" {
				if err = e.applyBatch([]Event{ev}, p); err != nil {
					return err
				}
			}
		}
		batches = append(batches, events)
		names = append(names, p)
	}
	for i, batch := range batches {
		if err = e.applyBatch(batch, names[i]); err != nil {
			e.reportError(err)
		} else if info, er := os.Stat(names[i]); er == nil {
			_, err = e.DB.Exec("INSERT OR REPLACE INTO segments VALUES(?,?,?)", names[i], info.Size(), info.ModTime().UnixNano())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func terms(text string) string {
	var out []string
	var word []rune
	var prev rune
	flush := func() {
		if len(word) > 0 {
			out = append(out, strings.ToLower(string(word)))
			word = nil
		}
	}
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			flush()
			out = append(out, string(r))
			if prev != 0 {
				out = append(out, string([]rune{prev, r}))
			}
			prev = r
		} else {
			prev = 0
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				word = append(word, r)
			} else {
				flush()
			}
		}
	}
	flush()
	return strings.Join(out, " ")
}
func (e *Engine) Sessions(agent string, offset, limit int) ([]Session, int, error) {
	where := ""
	var args []any
	if agent != "" {
		where = " WHERE agent=?"
		args = append(args, agent)
	}
	var count int
	if err := e.DB.QueryRow("SELECT count(*) FROM sessions"+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := e.DB.Query("SELECT data FROM sessions"+where+" ORDER BY updated DESC,id LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	var list []Session
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			rows.Close()
			return nil, 0, err
		}
		var s Session
		_ = json.Unmarshal(b, &s)
		list = append(list, s)
	}
	rows.Close()
	for i := range list {
		_ = e.DB.QueryRow("SELECT count(*) FROM messages WHERE session=?", list[i].ID).Scan(&list[i].Count)
	}
	return list, count, nil
}
func (e *Engine) Session(id string, offset, limit int) (Session, error) {
	var s Session
	var b []byte
	if err := e.DB.QueryRow("SELECT data FROM sessions WHERE id=?", id).Scan(&b); err != nil {
		return s, err
	}
	_ = json.Unmarshal(b, &s)
	_ = e.DB.QueryRow("SELECT count(*) FROM messages WHERE session=?", id).Scan(&s.Count)
	rows, err := e.DB.Query("SELECT version,data FROM messages WHERE session=? ORDER BY seq,id LIMIT ? OFFSET ?", id, limit, offset)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		var version string
		_ = rows.Scan(&version, &b)
		if err = json.Unmarshal(b, &m); err != nil {
			return s, err
		}
		m.Version = version
		s.Messages = append(s.Messages, m)
	}
	return s, rows.Err()
}
func (e *Engine) Memories(id string, history bool) ([]Memory, error) {
	query := `SELECT m.data FROM memories m WHERE 1=1`
	var args []any
	if id != "" {
		query += " AND m.id=?"
		args = append(args, id)
	}
	if !history {
		query += ` AND NOT EXISTS(SELECT 1 FROM parents p WHERE p.memory=m.id AND p.parent=m.revision)`
	}
	query += " ORDER BY m.updated DESC,m.revision"
	rows, err := e.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Memory
	for rows.Next() {
		var b []byte
		var m Memory
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

type SearchQuery struct {
	Query, Agent, Role, Kind, Since, Sort string
	Limit, Offset                         int
}

func (e *Engine) Search(q SearchQuery) ([]Hit, error) {
	tokens := queryTerms(q.Query)
	if len(tokens) == 0 {
		return []Hit{}, nil
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 30
	}
	var escaped []string
	seen := map[string]bool{}
	for _, t := range tokens {
		if !seen[t] {
			escaped = append(escaped, `"`+strings.ReplaceAll(t, `"`, `""`)+`"`)
			seen[t] = true
		}
	}
	query := `SELECT session,memory,message,title,agent,role,text,time FROM search WHERE search MATCH ?`
	args := []any{strings.Join(escaped, " AND ")}
	for _, f := range []struct{ col, val string }{{"agent", q.Agent}, {"role", q.Role}} {
		if f.val != "" {
			query += " AND " + f.col + "=?"
			args = append(args, f.val)
		}
	}
	if q.Kind == "memory" {
		query += " AND memory!=''"
	}
	if q.Kind == "message" {
		query += " AND memory=''"
	}
	if q.Since != "" {
		query += " AND time>=?"
		args = append(args, q.Since)
	}
	if q.Sort == "relevance" {
		query += " ORDER BY rank LIMIT ? OFFSET ?"
	} else {
		query += " ORDER BY rowid DESC LIMIT ? OFFSET ?"
	}
	args = append(args, q.Limit, q.Offset)
	rows, err := e.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Hit
	for rows.Next() {
		var h Hit
		if err = rows.Scan(&h.Session, &h.Memory, &h.Message, &h.Title, &h.Agent, &h.Role, &h.Text, &h.Time); err != nil {
			return nil, err
		}
		h.Text = excerpt(h.Text, q.Query, 500)
		list = append(list, h)
	}
	return list, rows.Err()
}
func excerpt(text, query string, n int) string {
	r := []rune(text)
	if len(r) <= n {
		return text
	}
	pos := strings.Index(strings.ToLower(text), strings.ToLower(query))
	start := 0
	if pos > 0 {
		start = len([]rune(text[:pos])) - 100
		if start < 0 {
			start = 0
		}
	}
	end := start + n
	if end > len(r) {
		end = len(r)
	}
	return string(r[start:end]) + "…"
}
func (e *Engine) Relations(id string) ([]Relation, error) {
	rows, err := e.DB.Query("SELECT origin,target,kind FROM relations WHERE origin=? OR target=?", id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Relation
	for rows.Next() {
		var r Relation
		_ = rows.Scan(&r.From, &r.To, &r.Kind)
		out = append(out, r)
	}
	return out, rows.Err()
}
func (e *Engine) Link(r Relation) error {
	if r.Kind != "continuation" && r.Kind != "branch" && r.Kind != "reference" {
		return fail("无效关联类型")
	}
	if r.From == r.To {
		return fail("不能关联自身")
	}
	for _, id := range []string{r.From, r.To} {
		if _, err := e.Session(id, 0, 0); err != nil {
			return err
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.persist([]Event{newEvent(e.Config.Device, "relation", r)})
}
func (e *Engine) Commit(m Memory) (Memory, error) {
	if m.Title == "" || m.Body == "" {
		return m, fail("标题与正文不能为空")
	}
	if m.Kind != "summary" && m.Kind != "preference" && m.Kind != "background" && m.Kind != "knowledge" {
		return m, fail("未知记忆类型")
	}
	if m.ID == "" {
		m.ID = randomID()
	}
	if err := checkID(m.ID); err != nil {
		return m, err
	}
	if e.Forgotten("memory", m.ID) {
		return m, fail("该记忆已被遗忘")
	}
	if m.Kind == "summary" && m.Session == "" {
		return m, fail("摘要缺少来源会话")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, p := range m.Parents {
		var n int
		_ = e.DB.QueryRow("SELECT count(*) FROM memories WHERE id=? AND revision=?", m.ID, p).Scan(&n)
		if n == 0 {
			return m, fail("父版本不存在")
		}
	}
	for _, src := range m.Evidence {
		var n int
		_ = e.DB.QueryRow("SELECT count(*) FROM messages WHERE session=? AND id=?", src.Session, src.Message).Scan(&n)
		if n == 0 {
			return m, fail("原文依据不存在")
		}
	}
	for _, id := range m.Covered {
		var n int
		_ = e.DB.QueryRow("SELECT count(*) FROM messages WHERE session=? AND version=?", m.Session, id).Scan(&n)
		if n == 0 {
			return m, fail("摘要覆盖范围包含未知消息版本")
		}
	}
	m.Updated = utc()
	if m.Author == "" {
		m.Author = "current-agent"
	}
	m.Revision = ""
	ev := newEvent(e.Config.Device, "memory", m)
	if err := e.persist([]Event{ev}); err != nil {
		return m, err
	}
	m.Revision = ev.ID
	return m, e.renderMemory(m)
}
func (e *Engine) renderMemory(m Memory) error {
	heads, err := e.Memories(m.ID, false)
	if err != nil {
		return err
	}
	if len(heads) > 1 {
		for _, h := range heads {
			if err = atomicWrite(filepath.Join(e.Config.Root, "memories", m.ID+"--"+h.Revision[:12]+".md"), memoryMarkdown(h), 0600); err != nil {
				return err
			}
		}
		return atomicWrite(filepath.Join(e.Config.Root, "memories", m.ID+".md"), []byte("# "+m.Title+"\n\n存在并发版本，请由 agent 合并。各版本位于同目录，原文依据未改变。\n"), 0600)
	}
	if len(heads) == 1 {
		return atomicWrite(filepath.Join(e.Config.Root, "memories", m.ID+".md"), memoryMarkdown(heads[0]), 0600)
	}
	return nil
}
func memoryMarkdown(m Memory) []byte {
	return []byte("<!-- memvault:" + string(encoded(m)) + " -->\n# " + m.Title + "\n\n" + m.Body + "\n")
}
func (e *Engine) WatchMemoryEdits() error {
	entries, err := os.ReadDir(filepath.Join(e.Config.Root, "memories"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		p := filepath.Join(e.Config.Root, "memories", entry.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := string(b)
		if !strings.HasPrefix(s, "<!-- memvault:") {
			continue
		}
		end := strings.Index(s, " -->")
		if end < 0 {
			continue
		}
		var m Memory
		if json.Unmarshal([]byte(s[len("<!-- memvault:"):end]), &m) != nil {
			continue
		}
		body := strings.TrimSpace(s[end+len(" -->"):])
		body = strings.TrimPrefix(body, "# "+m.Title)
		body = strings.TrimSpace(body)
		if body == strings.TrimSpace(m.Body) {
			continue
		}
		m.Parents = []string{m.Revision}
		m.Body = body
		m.Author = "file-edit"
		if _, err = e.Commit(m); err != nil {
			return err
		}
	}
	return nil
}
func (e *Engine) Context(sid, query string) (map[string]any, error) {
	var session Session
	var err error
	if sid != "" {
		session, err = e.Session(sid, 0, 0)
		if errors.Is(err, sql.ErrNoRows) {
			// A new session may not be archived yet; keep its identity so the agent can still maintain it.
			session = Session{ID: sid}
		} else if err != nil {
			return nil, err
		}
	}
	memories, err := e.Memories("", false)
	if err != nil {
		return nil, err
	}
	var core, summaries []Memory
	covered := map[string]bool{}
	first := true
	for _, m := range memories {
		if m.Session == sid && m.Kind == "summary" {
			summaries = append(summaries, m)
			current := map[string]bool{}
			for _, v := range m.Covered {
				current[v] = true
			}
			if first {
				covered = current
				first = false
			} else {
				for v := range covered {
					if !current[v] {
						delete(covered, v)
					}
				}
			}
		} else if m.Session == "" {
			core = append(core, m)
		}
	}
	var pending []map[string]any
	if sid != "" {
		rows, err := e.DB.Query("SELECT version,data FROM messages WHERE session=? ORDER BY seq DESC LIMIT 100", sid)
		if err != nil {
			return nil, err
		}
		budget := e.configSnapshot().ContextBudget
		if budget == 0 {
			budget = 16000
		}
		for rows.Next() {
			var version string
			var b []byte
			_ = rows.Scan(&version, &b)
			if covered[version] {
				continue
			}
			var m Message
			_ = json.Unmarshal(b, &m)
			r := []rune(m.Text)
			if budget <= 0 {
				break
			}
			truncated := false
			if len(r) > budget {
				m.Text = string(r[:budget])
				truncated = true
			}
			budget -= len([]rune(m.Text))
			pending = append(pending, map[string]any{"version": version, "message": m, "truncated": truncated})
		}
		rows.Close()
		for i, j := 0, len(pending)-1; i < j; i, j = i+1, j-1 {
			pending[i], pending[j] = pending[j], pending[i]
		}
	}
	related, _ := e.Search(SearchQuery{Query: query, Limit: 5})
	tasks, _ := e.Tasks()
	selectedTasks := make([]Task, 0, 3)
	for _, t := range tasks {
		if t.Session == sid {
			selectedTasks = append(selectedTasks, t)
			break
		}
	}
	for _, t := range tasks {
		if t.Session != sid && len(selectedTasks) < 3 {
			selectedTasks = append(selectedTasks, t)
		}
	}
	tasks = selectedTasks
	protocol := agentProtocol
	if b, er := os.ReadFile(filepath.Join(e.Config.Root, "AGENTS.md")); er == nil {
		protocol = string(b)
	}
	return map[string]any{"protocol": protocol, "session": session, "core": core, "summaries": summaries, "summary_conflict": len(summaries) > 1, "pending_messages": pending, "related": related, "maintenance": tasks}, nil
}
func (e *Engine) Tasks() ([]Task, error) {
	rows, err := e.DB.Query(`SELECT s.id,s.title,count(m.id) FROM sessions s JOIN messages m ON m.session=s.id GROUP BY s.id ORDER BY s.updated DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	for rows.Next() {
		var t Task
		_ = rows.Scan(&t.Session, &t.Title, &t.Pending)
		t.ID = "summarize-" + t.Session
		t.Kind = "summarize"
		tasks = append(tasks, t)
	}
	rows.Close()
	memories, _ := e.Memories("", false)
	covered := map[string]map[string]bool{}
	headCount := map[string]int{}
	for _, m := range memories {
		if m.Kind != "summary" {
			continue
		}
		headCount[m.Session]++
		if covered[m.Session] == nil {
			covered[m.Session] = map[string]bool{}
		}
		for _, v := range m.Covered {
			covered[m.Session][v] = true
		}
	}
	var out []Task
	for _, t := range tasks {
		if headCount[t.Session] > 1 {
			t.Kind = "merge"
		} else if len(covered[t.Session]) > 0 {
			t.Pending = 0
			r, er := e.DB.Query("SELECT version FROM messages WHERE session=?", t.Session)
			if er != nil {
				return nil, er
			}
			for r.Next() {
				var v string
				_ = r.Scan(&v)
				if !covered[t.Session][v] {
					t.Pending++
				}
			}
			r.Close()
		}
		if t.Pending > 0 || t.Kind == "merge" {
			_ = e.DB.QueryRow("SELECT owner,until_time FROM leases WHERE id=? AND until_time>?", t.ID, utc()).Scan(&t.Owner, &t.LeaseUntil)
			out = append(out, t)
		}
	}
	return out, nil
}
func (e *Engine) Claim(id, owner string) (Task, error) {
	if owner == "" {
		return Task{}, fail("缺少认领者")
	}
	until := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano)
	r, err := e.DB.Exec(`INSERT INTO leases VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET owner=excluded.owner,until_time=excluded.until_time WHERE leases.until_time<? OR leases.owner=?`, id, owner, until, utc(), owner)
	if err != nil {
		return Task{}, err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return Task{}, fail("维护事项已被其他 agent 认领")
	}
	return Task{ID: id, Owner: owner, LeaseUntil: until}, nil
}
func (e *Engine) Rebuild() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, err := e.DB.Exec(`DELETE FROM search_refs; DELETE FROM segments; DELETE FROM applied;DELETE FROM sessions;DELETE FROM messages;DELETE FROM memories;DELETE FROM parents;DELETE FROM relations;DELETE FROM search;DELETE FROM cursors;`); err != nil {
		return err
	}
	return e.Replay()
}
func (e *Engine) ReadableSession(id string) (string, error) {
	s, err := e.Session(id, 0, 1000000)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n来源：%s\n会话：%s\n归档范围：%s\n\n", s.Title, s.Agent, s.ID, s.Coverage)
	for _, m := range s.Messages {
		fmt.Fprintf(&b, "## %s · %s\n\n%s\n\n", m.Role, m.Time, m.Text)
		for _, a := range m.Attachments {
			if a.Missing {
				fmt.Fprintf(&b, "附件：%s（尚未取得副本）\n\n", a.Name)
			} else {
				fmt.Fprintf(&b, "附件：[%s](../objects/%s/%s) · SHA-256 %s\n\n", strings.ReplaceAll(a.Name, "]", ""), a.Blob[:2], a.Blob, a.Blob)
			}
		}
	}
	return b.String(), nil
}
func (e *Engine) Materialize(id string) error {
	body, err := e.ReadableSession(id)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(e.Config.Root, "sessions", id+".md"), []byte(body), 0600)
}
func (e *Engine) Diagnose() map[string]any {
	e.statusMu.RLock()
	sources := make([]SourceStatus, 0, len(e.Statuses))
	for _, s := range e.Statuses {
		sources = append(sources, s)
	}
	issues := append([]string{}, e.ScanErrors...)
	if _, err := os.Stat(filepath.Join(e.Config.Root, "memvault.json")); err != nil {
		issues = append(issues, vaultUnavailableIssue(err))
	}
	e.statusMu.RUnlock()
	sort.Slice(sources, func(i, j int) bool { return sources[i].Agent < sources[j].Agent })
	var sessions, messages, memories int
	_ = e.DB.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions)
	_ = e.DB.QueryRow("SELECT count(*) FROM messages").Scan(&messages)
	_ = e.DB.QueryRow("SELECT count(DISTINCT id) FROM memories").Scan(&memories)
	return map[string]any{"version": Version, "root": e.Config.Root, "sessions": sessions, "messages": messages, "memories": memories, "sources": sources, "issues": issues, "time": utc()}
}

// vaultUnavailableIssue explains why the vault cannot be read, naming the fix
// when the operating system, not the disk, is refusing access.
func vaultUnavailableIssue(err error) string {
	const pending = "新资料暂存本机，恢复后自动补写，尚未确认归档到记忆目录"
	if os.IsPermission(err) {
		if runtime.GOOS == "darwin" {
			return "系统拒绝后台访问记忆目录；请在“系统设置 › 隐私与安全性 › 完全磁盘访问权限”中加入 memvault 后运行 memvault repair --action service。" + pending
		}
		return "没有权限读取记忆目录；请检查目录权限。" + pending
	}
	return "记忆目录暂不可用；" + pending
}
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}

func (e *Engine) configSnapshot() Config {
	e.configMu.RLock()
	defer e.configMu.RUnlock()
	c := e.Config
	c.Sources = append([]Source(nil), c.Sources...)
	return c
}

// Search CJK runs by overlapping bigrams; single-character runs retain unigram support.
// Indexing also retains unigrams, but joining both repeats redundant constraints.
func queryTerms(text string) []string {
	var result []string
	var run []rune
	flush := func() {
		if len(run) == 1 {
			result = append(result, string(run))
		} else {
			for i := 1; i < len(run); i++ {
				result = append(result, string(run[i-1:i+1]))
			}
		}
		run = nil
	}
	var latin []rune
	flushLatin := func() {
		if len(latin) > 0 {
			result = append(result, strings.ToLower(string(latin)))
			latin = nil
		}
	}
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			flushLatin()
			run = append(run, r)
		} else {
			flush()
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				latin = append(latin, r)
			} else {
				flushLatin()
			}
		}
	}
	flush()
	flushLatin()
	return result
}
