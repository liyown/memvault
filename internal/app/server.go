package app

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed all:ui
var uiFiles embed.FS

//go:embed THIRD_PARTY_NOTICES.txt
var thirdPartyNotices string

type Server struct {
	Engine      *Engine
	Address     string
	inboxMu     sync.Mutex
	listenersMu sync.Mutex
	listeners   map[chan struct{}]bool
	shutdown    context.CancelFunc
}

func NewServer(e *Engine, address string) *Server {
	return &Server{Engine: e, Address: address, listeners: map[chan struct{}]bool{}}
}
func (s *Server) authorized(r *http.Request) bool {
	if s.Engine.Config.Token == "" {
		return false
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if cookie, err := r.Cookie("memvault_session"); err == nil && token == "" {
		token = cookie.Value
	}
	return token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.Engine.Config.Token)) == 1
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", s.api)
	static, _ := fs.Sub(uiFiles, "ui")
	files := http.FileServer(http.FS(static))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.URL.Path != "/" {
			if _, err := fs.Stat(static, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		files.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		host := r.Host
		expected := s.Address
		if host != expected {
			http.Error(w, "不允许的主机", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+expected {
			http.Error(w, "不允许的来源", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && !s.authorized(r) {
			http.Error(w, "请使用 memvault open 打开本地页面", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
func response(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, os.ErrNotExist) {
			code = 404
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}
func decode(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 16<<20)).Decode(v)
}
func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "不支持此请求方式", http.StatusMethodNotAllowed)
	return false
}
func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	e := s.Engine
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	q := r.URL.Query()
	switch {
	case path == "relocate":
		if !requireMethod(w, r, "POST") {
			return
		}
		var p struct {
			Root string `json:"root"`
		}
		if err := decode(r, &p); err != nil {
			response(w, nil, err)
			return
		}
		response(w, object{"accepted": true, "note": "正在复制并校验档案；完成后刷新页面。旧目录保留。"}, launchSetup(e.configSnapshot(), p.Root))
	case path == "pick":
		if !requireMethod(w, r, "POST") {
			return
		}
		var p pickRequest
		if err := decode(r, &p); err != nil {
			response(w, nil, err)
			return
		}
		picked, err := pickPath(r.Context(), p)
		switch {
		case errors.Is(err, errPickCancelled):
			response(w, object{"cancelled": true}, nil)
		case errors.Is(err, errPickUnsupported):
			response(w, object{"unsupported": true}, nil)
		default:
			response(w, object{"path": picked}, err)
		}
	case path == "browse":
		if !requireMethod(w, r, "GET") {
			return
		}
		v, err := browseDir(q.Get("path"), q.Get("files") == "1")
		response(w, v, err)
	case path == "shutdown":
		if !requireMethod(w, r, "POST") {
			return
		}
		response(w, object{"ok": true}, nil)
		if s.shutdown != nil {
			go func() { time.Sleep(200 * time.Millisecond); s.shutdown() }()
		}
	case path == "auth":
		if !requireMethod(w, r, "POST") {
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "memvault_session", Value: e.Config.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		response(w, map[string]any{"ok": true}, nil)
	case path == "status":
		if !requireMethod(w, r, "GET") {
			return
		}
		status := e.Diagnose()
		status["integrations"] = IntegrationStatus(e.configSnapshot())
		response(w, status, nil)
	case path == "sessions":
		if !requireMethod(w, r, "GET") {
			return
		}
		limit := parseInt(q.Get("limit"), 50)
		if limit > 200 {
			limit = 200
		}
		items, count, err := e.Sessions(q.Get("agent"), parseInt(q.Get("offset"), 0), limit)
		response(w, map[string]any{"items": items, "total": count}, err)
	case strings.HasPrefix(path, "sessions/"):
		if !requireMethod(w, r, "GET") {
			return
		}
		id := strings.TrimPrefix(path, "sessions/")
		if q.Get("format") == "markdown" {
			body, err := e.ReadableSession(id)
			if err != nil {
				response(w, nil, err)
				return
			}
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.md"`)
			_, _ = io.WriteString(w, body)
			return
		}
		offset := parseInt(q.Get("offset"), 0)
		if mid := q.Get("message"); mid != "" {
			var seq int64
			if err := e.DB.QueryRow("SELECT seq FROM messages WHERE session=? AND id=?", id, mid).Scan(&seq); err == nil {
				_ = e.DB.QueryRow("SELECT count(*) FROM messages WHERE session=? AND seq<?", id, seq).Scan(&offset)
				if offset > 5 {
					offset -= 5
				}
			}
		}
		limit := parseInt(q.Get("limit"), 100)
		if limit > 500 {
			limit = 500
		}
		session, err := e.Session(id, offset, limit)
		relations, _ := e.Relations(id)
		memories, _ := e.Memories("", false)
		var summary []Memory
		for _, m := range memories {
			if m.Session == id && m.Kind == "summary" {
				summary = append(summary, m)
			}
		}
		response(w, map[string]any{"session": session, "relations": relations, "summaries": summary, "offset": offset}, err)
	case path == "search":
		if !requireMethod(w, r, "GET") {
			return
		}
		hits, err := e.Search(SearchQuery{Query: q.Get("q"), Agent: q.Get("agent"), Role: q.Get("role"), Kind: q.Get("kind"), Since: q.Get("since"), Sort: q.Get("sort"), Limit: parseInt(q.Get("limit"), 30), Offset: parseInt(q.Get("offset"), 0)})
		response(w, map[string]any{"items": hits}, err)
	case path == "memories":
		if !requireMethod(w, r, "GET", "POST") {
			return
		}
		if r.Method == "GET" {
			items, err := e.Memories(q.Get("id"), q.Get("history") == "true")
			response(w, map[string]any{"items": items}, err)
		} else {
			var m Memory
			if err := decode(r, &m); err != nil {
				response(w, nil, err)
				return
			}
			value, err := e.Commit(m)
			response(w, value, err)
		}
	case path == "context":
		if !requireMethod(w, r, "GET") {
			return
		}
		sid := q.Get("session")
		if sid == "" && q.Get("native_id") != "" {
			sid = key(q.Get("agent"), q.Get("native_id"))
		}
		value, err := e.Context(sid, q.Get("q"))
		if s, ok := value["session"].(Session); ok && s.NativeID == "" && q.Get("native_id") != "" {
			s.NativeID, s.Agent = q.Get("native_id"), q.Get("agent")
			value["session"] = s
		}
		response(w, value, err)
	case path == "tasks":
		if !requireMethod(w, r, "GET", "POST") {
			return
		}
		if r.Method == "GET" {
			tasks, err := e.Tasks()
			response(w, map[string]any{"items": tasks}, err)
		} else {
			var p struct {
				ID    string `json:"id"`
				Owner string `json:"owner"`
			}
			if err := decode(r, &p); err != nil {
				response(w, nil, err)
				return
			}
			task, err := e.Claim(p.ID, p.Owner)
			response(w, task, err)
		}
	case path == "relations":
		if !requireMethod(w, r, "POST") {
			return
		}
		var rel Relation
		if err := decode(r, &rel); err != nil {
			response(w, nil, err)
			return
		}
		err := e.Link(rel)
		response(w, map[string]any{"ok": err == nil}, err)
	case path == "capture":
		if !requireMethod(w, r, "POST") {
			return
		}
		var p object
		if err := decode(r, &p); err != nil {
			response(w, nil, err)
			return
		}
		// Durable local inbox: hooks are acknowledged only after their envelope is saved.
		id := randomID()
		err := atomicWrite(filepath.Join(stateDir(), "inbox", id+".json"), encoded(p), 0600)
		if err == nil {
			go s.processInbox(context.Background())
		}
		response(w, map[string]any{"accepted": err == nil}, err)
	case path == "scan":
		if !requireMethod(w, r, "POST") {
			return
		}
		go func() { e.reportError(e.ProtectedScan(context.Background())) }()
		response(w, map[string]any{"accepted": true}, nil)
	case path == "events":
		if !requireMethod(w, r, "GET") {
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "不支持实时状态", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "keep-alive")
		ch := make(chan struct{}, 1)
		s.listenersMu.Lock()
		s.listeners[ch] = true
		s.listenersMu.Unlock()
		defer func() { s.listenersMu.Lock(); delete(s.listeners, ch); s.listenersMu.Unlock() }()
		tick := time.NewTicker(20 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ch:
				if _, err := fmt.Fprint(w, "event: change\ndata: {}\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case <-tick.C:
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	case path == "export":
		if !requireMethod(w, r, "GET") {
			return
		}
		file, err := os.CreateTemp(stateDir(), "export-*.zip")
		if err != nil {
			response(w, nil, err)
			return
		}
		name := file.Name()
		defer func() { _ = os.Remove(name) }()
		err = e.Export(file)
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			response(w, nil, err)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="memvault-archive.zip"`)
		w.Header().Set("Content-Type", "application/zip")
		http.ServeFile(w, r, name)
	case path == "import":
		if !requireMethod(w, r, "POST") {
			return
		}
		file, err := os.CreateTemp(stateDir(), "import-upload-*.zip")
		if err != nil {
			response(w, nil, err)
			return
		}
		name := file.Name()
		defer func() { _ = os.Remove(name) }()
		_, err = io.Copy(file, http.MaxBytesReader(w, r.Body, 32<<30))
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = e.Import(name)
		}
		response(w, map[string]any{"ok": err == nil}, err)
	case path == "forget/preview":
		if !requireMethod(w, r, "GET") {
			return
		}
		value, err := e.ForgetPreview(q.Get("kind"), q.Get("id"))
		response(w, value, err)
	case path == "forget":
		if !requireMethod(w, r, "POST") {
			return
		}
		var p struct {
			Kind     string `json:"kind"`
			ID       string `json:"id"`
			Explicit bool   `json:"explicit"`
		}
		if err := decode(r, &p); err != nil {
			response(w, nil, err)
			return
		}
		err := e.Forget(p.Kind, p.ID, p.Explicit)
		response(w, map[string]any{"ok": err == nil}, err)
	case path == "resume":
		if !requireMethod(w, r, "POST") {
			return
		}
		var p struct {
			Session string `json:"session"`
			Agent   string `json:"agent"`
			Launch  bool   `json:"launch"`
		}
		if err := decode(r, &p); err != nil {
			response(w, nil, err)
			return
		}
		value, err := e.Resume(p.Session, p.Agent, p.Launch)
		response(w, value, err)
	case path == "settings":
		if !requireMethod(w, r, "GET", "POST") {
			return
		}
		if r.Method == "GET" {
			response(w, map[string]any{"root": e.Config.Root, "sources": e.configSnapshot().Sources, "context_budget": e.configSnapshot().ContextBudget, "version": Version}, nil)
		} else {
			var p struct {
				Sources []Source `json:"sources"`
				Budget  int      `json:"context_budget"`
			}
			if err := decode(r, &p); err != nil {
				response(w, nil, err)
				return
			}
			for _, source := range p.Sources {
				if source.Agent != "codex" && source.Agent != "claude" && source.Agent != "opencode" {
					response(w, nil, fail("不支持的来源"))
					return
				}
			}
			e.configMu.Lock()
			old := e.Config
			e.Config.Sources = p.Sources
			if p.Budget >= 1024 && p.Budget <= 100000 {
				e.Config.ContextBudget = p.Budget
			}
			err := saveConfig(e.Config)
			if err != nil {
				e.Config = old
			}
			e.configMu.Unlock()
			response(w, map[string]any{"ok": err == nil}, err)
		}
	case path == "repair":
		if !requireMethod(w, r, "POST") {
			return
		}
		var p struct {
			Action string `json:"action"`
		}
		if err := decode(r, &p); err != nil {
			response(w, nil, err)
			return
		}
		if p.Action == "verify" {
			response(w, map[string]any{"issues": e.Verify()}, nil)
		} else if p.Action == "rebuild" {
			err := e.Rebuild()
			response(w, map[string]any{"ok": err == nil}, err)
		} else if p.Action == "integrations" {
			value, err := Configure(e.configSnapshot(), false)
			for _, item := range value {
				if item.Error != "" {
					err = fail("%s 接入需要处理：%s", item.Agent, item.Error)
					break
				}
			}
			response(w, value, err)
		} else if p.Action == "service" {
			if err := serviceInstall(e.configSnapshot(), false); err != nil {
				response(w, nil, err)
			} else {
				response(w, map[string]any{"ok": true}, nil)
			}
		} else {
			response(w, nil, fail("未知修复操作"))
		}
	case strings.HasPrefix(path, "objects/"):
		if !requireMethod(w, r, "GET") {
			return
		}
		id := strings.TrimPrefix(path, "objects/")
		b, err := e.BlobRead(id)
		if err != nil {
			response(w, nil, err)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(first(q.Get("name"), id)))
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(b)
	default:
		http.NotFound(w, r)
	}
}
func (s *Server) processInbox(ctx context.Context) {
	if !s.inboxMu.TryLock() {
		return
	}
	defer s.inboxMu.Unlock()
	files, _ := filepath.Glob(filepath.Join(stateDir(), "inbox", "*.json"))
	for _, file := range files {
		var p object
		if err := readJSON(file, &p); err != nil {
			s.Engine.reportError(err)
			continue
		}
		agent := str(p["agent"])
		path := str(p["transcript_path"])
		var err error
		_, rootErr := os.Stat(filepath.Join(s.Engine.Config.Root, "memvault.json"))
		if rootErr == nil && path != "" && (agent == "codex" || agent == "claude") {
			if _, er := os.Stat(path); os.IsNotExist(er) {
				// The platform removed or never wrote this transcript; whatever was archived stays archived.
				_ = os.Remove(file)
				continue
			}
			err = s.Engine.CaptureFile(agent, path, true)
		} else {
			err = s.Engine.ProtectedScan(ctx)
		}
		if err != nil {
			s.Engine.reportError(err)
			continue
		}
		_ = os.Remove(file)
	}
}
func (s *Server) broadcast(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.Engine.changed:
			s.listenersMu.Lock()
			for ch := range s.listeners {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
			s.listenersMu.Unlock()
		}
	}
}

// errAlreadyRunning means another instance holds the service lock.
var errAlreadyRunning = errors.New("本机服务已经在运行")

func Serve(ctx context.Context, c Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	unlock, err := lockInstance(filepath.Join(stateDir(), "service.lock"))
	if errors.Is(err, errAlreadyRunning) && os.Getenv("MEMVAULT_MANAGED") == "1" {
		// A login agent racing a running instance must exit cleanly; a failure
		// exit makes launchd/systemd restart it every few seconds forever.
		return nil
	}
	if err != nil {
		return err
	}
	defer unlock()
	_ = atomicWrite(filepath.Join(stateDir(), "startup.json"), encoded(object{"pid": os.Getpid(), "time": utc(), "stage": "opening_vault_and_index", "managed": os.Getenv("MEMVAULT_MANAGED") == "1"}), 0600)
	e, err := OpenEngine(c)
	if err != nil {
		return err
	}
	defer func() { _ = e.Close() }()
	address := c.Address
	if address == "" {
		address = "127.0.0.1:27941"
	}
	host, _, er := net.SplitHostPort(address)
	if er != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fail("服务只允许使用本机回环地址")
	}
	if c.Token == "" {
		return fail("缺少本机访问凭据，请重新 setup")
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return err
	}
	defer func() { _ = ln.Close() }()
	address = ln.Addr().String()
	c.Address = address
	e.Config.Address = address
	if err = saveConfig(c); err != nil {
		return err
	}
	_ = atomicWrite(filepath.Join(stateDir(), "startup.json"), encoded(object{"pid": os.Getpid(), "time": utc(), "stage": "listening", "managed": os.Getenv("MEMVAULT_MANAGED") == "1"}), 0600)
	s := NewServer(e, address)
	s.shutdown = cancel
	server := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	go e.Background(ctx)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.processInbox(ctx)
			}
		}
	}()
	go s.broadcast(ctx)
	go s.processInbox(ctx)
	go func() {
		<-ctx.Done()
		timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(timeout)
	}()
	err = server.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
