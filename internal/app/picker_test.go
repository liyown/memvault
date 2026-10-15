package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func apiCall(t *testing.T, s *Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(method, "http://127.0.0.1:27941/api/v1/"+path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	var v map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	return w.Code, v
}

func TestBrowseListsDirectoriesOnly(t *testing.T) {
	s := NewServer(testEngine(t), "127.0.0.1:27941")
	dir := t.TempDir()
	must(t, os.Mkdir(filepath.Join(dir, "Beta"), 0o755))
	must(t, os.Mkdir(filepath.Join(dir, "alpha"), 0o755))
	must(t, os.Mkdir(filepath.Join(dir, ".hidden"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644))

	code, v := apiCall(t, s, "GET", "browse?path="+dir, "")
	if code != 200 {
		t.Fatalf("status %d %v", code, v)
	}
	entries := v["entries"].([]any)
	var got []string
	for _, e := range entries {
		got = append(got, e.(map[string]any)["name"].(string))
	}
	if strings.Join(got, ",") != ".hidden,alpha,Beta" {
		t.Fatalf("entries %v", got)
	}
	if v["parent"] != filepath.Dir(dir) {
		t.Fatalf("parent %v", v["parent"])
	}

	_, v = apiCall(t, s, "GET", "browse?files=1&path="+dir, "")
	if len(v["entries"].([]any)) != 4 {
		t.Fatalf("files mode %v", v["entries"])
	}
	if code, _ := apiCall(t, s, "GET", "browse?path=relative/dir", ""); code != 400 {
		t.Fatalf("relative path accepted: %d", code)
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:27941/api/v1/browse", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("unauthenticated browse %d", w.Code)
	}
}

func TestPickOutcomes(t *testing.T) {
	s := NewServer(testEngine(t), "127.0.0.1:27941")
	oldRun, oldLook := runPick, lookPath
	t.Cleanup(func() { runPick, lookPath = oldRun, oldLook })
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	picked := t.TempDir()

	runPick = func(ctx context.Context, env []string, name string, args ...string) (string, error) {
		return picked + "/\n", nil
	}
	if _, v := apiCall(t, s, "POST", "pick", `{"kind":"folder","prompt":"选"}`); v["path"] != picked {
		t.Fatalf("picked %v", v)
	}
	if !pickWasCancelled("osascript", &exec.ExitError{Stderr: []byte("execution error: 用户已取消。 (-128)")}) {
		t.Fatal("osascript -128 should be a cancel")
	}
	if pickWasCancelled("osascript", &exec.ExitError{Stderr: []byte("Not authorized to send Apple events (-1743)")}) {
		t.Fatal("osascript -1743 is not a cancel")
	}
	runPick = func(ctx context.Context, env []string, name string, args ...string) (string, error) {
		return "\n", nil
	}
	if _, v := apiCall(t, s, "POST", "pick", `{}`); v["cancelled"] != true {
		t.Fatalf("cancel %v", v)
	}
	runPick = func(ctx context.Context, env []string, name string, args ...string) (string, error) {
		return "", &exec.ExitError{}
	}
	if _, v := apiCall(t, s, "POST", "pick", `{}`); v["unsupported"] != true {
		t.Fatalf("dialog failure should fall back %v", v)
	}
	runPick = func(ctx context.Context, env []string, name string, args ...string) (string, error) {
		return "", errors.New("no display")
	}
	if _, v := apiCall(t, s, "POST", "pick", `{}`); v["unsupported"] != true {
		t.Fatalf("unsupported %v", v)
	}
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	if _, v := apiCall(t, s, "POST", "pick", `{}`); v["unsupported"] != true {
		t.Fatalf("missing tool %v", v)
	}
}

func TestAppleStringEscapes(t *testing.T) {
	if got := appleString(`a"b\c`); got != `"a\"b\\c"` {
		t.Fatal(got)
	}
}
