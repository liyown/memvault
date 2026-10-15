package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestManagedServeExitsCleanlyWhenAlreadyRunning(t *testing.T) {
	t.Setenv("MEMVAULT_HOME", t.TempDir())
	unlock, err := lockInstance(filepath.Join(stateDir(), "service.lock"))
	must(t, err)
	defer unlock()

	c := Config{Root: t.TempDir(), Token: "t", Address: "127.0.0.1:0"}
	if err := Serve(context.Background(), c); !errors.Is(err, errAlreadyRunning) {
		t.Fatalf("unmanaged serve: %v", err)
	}
	t.Setenv("MEMVAULT_MANAGED", "1")
	if err := Serve(context.Background(), c); err != nil {
		t.Fatalf("managed serve should exit cleanly: %v", err)
	}
}
