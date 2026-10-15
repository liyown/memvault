package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// pickRunner runs a native dialog command and returns its stdout. Tests replace it.
type pickRunner func(ctx context.Context, env []string, name string, args ...string) (string, error)

var (
	errPickUnsupported = errors.New("本机无法打开系统选择框")
	errPickCancelled   = errors.New("已取消选择")
	pickMu             sync.Mutex
	runPick            pickRunner = func(ctx context.Context, env []string, name string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.Output()
		return string(out), err
	}
	lookPath = exec.LookPath
)

type pickRequest struct {
	Kind   string `json:"kind"`
	Start  string `json:"start"`
	Prompt string `json:"prompt"`
}

// pickPath opens the platform's native folder or file chooser.
func pickPath(ctx context.Context, p pickRequest) (string, error) {
	if !pickMu.TryLock() {
		return "", fail("已有一个选择框正在打开")
	}
	defer pickMu.Unlock()
	files := p.Kind == "file"
	start := nearestDir(p.Start)
	prompt := p.Prompt
	if prompt == "" {
		prompt = "选择文件夹"
	}
	var name string
	var args, env []string
	switch runtime.GOOS {
	case "darwin":
		verb := "choose folder"
		if files {
			verb = "choose file"
		}
		// A bare osascript process cannot host a dialog; System Events brings it to the front.
		script := "tell application \"System Events\"\nactivate\nset p to POSIX path of (" + verb +
			" with prompt " + appleString(prompt) + " default location (POSIX file " + appleString(start) +
			"))\nend tell\nreturn p"
		name, args = "osascript", []string{"-e", script}
	case "windows":
		dialog := "$d = New-Object System.Windows.Forms.FolderBrowserDialog; $d.Description = $env:MEMVAULT_PICK_PROMPT; $d.SelectedPath = $env:MEMVAULT_PICK_START; $d.ShowNewFolderButton = $true; if ($d.ShowDialog() -eq 'OK') { $d.SelectedPath }"
		if files {
			dialog = "$d = New-Object System.Windows.Forms.OpenFileDialog; $d.Title = $env:MEMVAULT_PICK_PROMPT; $d.InitialDirectory = $env:MEMVAULT_PICK_START; if ($d.ShowDialog() -eq 'OK') { $d.FileName }"
		}
		env = []string{"MEMVAULT_PICK_PROMPT=" + prompt, "MEMVAULT_PICK_START=" + start}
		name, args = "powershell", []string{"-NoProfile", "-STA", "-Command", "Add-Type -AssemblyName System.Windows.Forms; " + dialog}
	default:
		if _, err := lookPath("zenity"); err == nil {
			name, args = "zenity", []string{"--file-selection", "--title=" + prompt, "--filename=" + strings.TrimSuffix(start, "/") + "/"}
			if !files {
				args = append(args, "--directory")
			}
		} else if _, err := lookPath("kdialog"); err == nil {
			mode := "--getexistingdirectory"
			if files {
				mode = "--getopenfilename"
			}
			name, args = "kdialog", []string{mode, start, "--title", prompt}
		} else {
			return "", errPickUnsupported
		}
	}
	if _, err := lookPath(name); err != nil {
		return "", errPickUnsupported
	}
	out, err := runPick(ctx, env, name, args...)
	path := strings.TrimSpace(out)
	if err == nil && path == "" {
		return "", errPickCancelled
	}
	if err != nil {
		if pickWasCancelled(name, err) {
			return "", errPickCancelled
		}
		// Missing display, denied automation permission and similar: use the in-app browser.
		return "", errPickUnsupported
	}
	return filepath.Clean(path), nil
}

// pickWasCancelled tells a user cancel apart from a dialog that could not open.
func pickWasCancelled(name string, err error) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return false
	}
	if name == "osascript" {
		// -128 is "User canceled"; other codes (e.g. -1743 not authorized) mean no dialog was shown.
		return strings.Contains(string(exit.Stderr), "(-128)")
	}
	// zenity and kdialog exit 1 on cancel; other codes are failures.
	return exit.ExitCode() == 1
}

// appleString quotes a value as an AppleScript string literal.
func appleString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// nearestDir returns the closest existing directory for a start location.
func nearestDir(p string) string {
	if p == "" || !filepath.IsAbs(p) {
		return home()
	}
	for p = filepath.Clean(p); ; p = filepath.Dir(p) {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
		if filepath.Dir(p) == p {
			return home()
		}
	}
}

type browseEntry struct {
	Name   string `json:"name"`
	Dir    bool   `json:"dir"`
	Hidden bool   `json:"hidden"`
}

// browseDir lists names under an absolute directory without reading file contents.
func browseDir(p string, files bool) (object, error) {
	if p == "" {
		p = home()
	}
	if !filepath.IsAbs(p) {
		return nil, fail("请使用完整路径")
	}
	p = filepath.Clean(p)
	info, err := os.Stat(p)
	if err != nil {
		return nil, fail("无法打开这个位置：%s", p)
	}
	if !info.IsDir() {
		p = filepath.Dir(p)
	}
	list, err := os.ReadDir(p)
	if err != nil {
		return nil, fail("没有权限读取这个位置：%s", p)
	}
	entries := []browseEntry{}
	for _, d := range list {
		dir := d.IsDir()
		if d.Type()&os.ModeSymlink != 0 {
			if target, err := os.Stat(filepath.Join(p, d.Name())); err == nil {
				dir = target.IsDir()
			}
		}
		if !dir && !files {
			continue
		}
		entries = append(entries, browseEntry{Name: d.Name(), Dir: dir, Hidden: strings.HasPrefix(d.Name(), ".")})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	parent := filepath.Dir(p)
	if parent == p {
		parent = ""
	}
	return object{"path": p, "parent": parent, "home": home(), "separator": string(filepath.Separator), "entries": entries}, nil
}
