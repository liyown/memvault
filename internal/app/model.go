package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

var Version = "0.1.0-preview"
const Schema = 1

type Config struct {
	Root          string   `json:"root"`
	Device        string   `json:"device"`
	Token         string   `json:"token"`
	Address       string   `json:"address"`
	Binary        string   `json:"binary"`
	Sources       []Source `json:"sources"`
	ContextBudget int      `json:"context_budget"`
}
type Source struct {
	Agent   string `json:"agent"`
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}
type SourceStatus struct {
	Agent    string `json:"agent"`
	State    string `json:"state"`
	Sessions int    `json:"sessions"`
	LastScan string `json:"last_scan,omitempty"`
	Error    string `json:"error,omitempty"`
}
type Attachment struct {
	Name    string `json:"name"`
	Blob    string `json:"blob,omitempty"`
	URL     string `json:"url,omitempty"`
	Size    int64  `json:"size,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}
type Message struct {
	Version     string       `json:"version,omitempty"`
	ID          string       `json:"id"`
	Role        string       `json:"role"`
	Text        string       `json:"text"`
	Time        string       `json:"time,omitempty"`
	Order       int64        `json:"order"`
	Revision    int64        `json:"revision,omitempty"`
	Complete    bool         `json:"complete"`
	Phase       string       `json:"phase,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}
type Session struct {
	ID       string    `json:"id"`
	NativeID string    `json:"native_id"`
	Agent    string    `json:"agent"`
	Title    string    `json:"title"`
	Created  string    `json:"created,omitempty"`
	Updated  string    `json:"updated,omitempty"`
	Parent   string    `json:"parent,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	Raw      []string  `json:"raw,omitempty"`
	Coverage string    `json:"coverage"`
	Issues   []string  `json:"issues,omitempty"`
	Count    int       `json:"count"`
	Missing  int       `json:"missing"`
}
type Evidence struct {
	Session string `json:"session"`
	Message string `json:"message"`
}
type Memory struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Kind     string     `json:"kind"`
	Body     string     `json:"body"`
	Session  string     `json:"session,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty"`
	Covered  []string   `json:"covered,omitempty"`
	Parents  []string   `json:"parents"`
	Revision string     `json:"revision"`
	Updated  string     `json:"updated"`
	Author   string     `json:"author"`
	Pending  bool       `json:"pending,omitempty"`
}
type Relation struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}
type Event struct {
	Schema  int             `json:"schema"`
	ID      string          `json:"id"`
	Device  string          `json:"device"`
	Kind    string          `json:"kind"`
	Time    string          `json:"time"`
	Payload json.RawMessage `json:"payload"`
}
type Change struct {
	Session string   `json:"session,omitempty"`
	Message *Message `json:"message,omitempty"`
	Meta    *Session `json:"meta,omitempty"`
}
type Tombstone struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}
type Task struct {
	ID         string `json:"id"`
	Session    string `json:"session"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	Pending    int    `json:"pending"`
	Owner      string `json:"owner,omitempty"`
	LeaseUntil string `json:"lease_until,omitempty"`
}
type Hit struct {
	Session string `json:"session,omitempty"`
	Memory  string `json:"memory,omitempty"`
	Message string `json:"message,omitempty"`
	Title   string `json:"title"`
	Agent   string `json:"agent,omitempty"`
	Role    string `json:"role,omitempty"`
	Text    string `json:"text"`
	Time    string `json:"time,omitempty"`
}

func utc() string          { return time.Now().UTC().Format(time.RFC3339Nano) }
func hash(b []byte) string { x := sha256.Sum256(b); return hex.EncodeToString(x[:]) }
func encoded(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func key(agent, id string) string { return hash([]byte(agent + ":" + id))[:32] }
func randomID() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func checkID(id string) error {
	if !validID.MatchString(id) {
		return errors.New("无效的记录编号")
	}
	return nil
}
func home() string {
	if p := os.Getenv("MEMVAULT_AGENT_HOME"); p != "" {
		return p
	}
	p, _ := os.UserHomeDir()
	return p
}
func configDir() string {
	if p := os.Getenv("MEMVAULT_HOME"); p != "" {
		return p
	}
	p, e := os.UserConfigDir()
	if e != nil {
		p = filepath.Join(home(), ".config")
	}
	return filepath.Join(p, "memvault")
}
func stateDir() string   { return filepath.Join(configDir(), "runtime") }
func configPath() string { return filepath.Join(configDir(), "config.json") }
func defaultSources() []Source {
	codex := os.Getenv("CODEX_HOME")
	if codex == "" {
		codex = filepath.Join(home(), ".codex")
	}
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	if claude == "" {
		claude = filepath.Join(home(), ".claude")
	}
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		xdg = filepath.Join(home(), ".local", "share")
	}
	return []Source{{"codex", codex, true}, {"claude", filepath.Join(claude, "projects"), true}, {"opencode", filepath.Join(xdg, "opencode", "opencode.db"), true}}
}
func loadConfig() (Config, error) {
	var c Config
	b, e := os.ReadFile(configPath())
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(b, &c)
	if c.ContextBudget < 1024 {
		c.ContextBudget = 16000
	}
	return c, e
}
func saveConfig(c Config) error { return atomicWrite(configPath(), encoded(c), 0600) }
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".writing-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if e = os.Rename(tmp, path); e != nil {
		return e
	}
	// Flush directory entries on systems that support it.
	if runtime.GOOS != "windows" {
		if d, e := os.Open(filepath.Dir(path)); e == nil {
			e = d.Sync()
			d.Close()
			if e != nil {
				return e
			}
		}
	}
	return nil
}
func copyFile(src, dst string) error {
	f, e := os.Open(src)
	if e != nil {
		return e
	}
	defer f.Close()
	b, e := io.ReadAll(f)
	if e != nil {
		return e
	}
	return atomicWrite(dst, b, 0600)
}
func readJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func pathInside(root, path string) bool {
	rel, e := filepath.Rel(root, path)
	return e == nil && rel != ".." && len(rel) > 0 && !filepath.IsAbs(rel) && !regexp.MustCompile(`^\.\.[/\\]`).MatchString(rel)
}
func fail(format string, a ...any) error { return fmt.Errorf(format, a...) }
