//go:build !windows

package kube

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeDocker logs `docker save` and `docker exec`, writing what exec read
// from stdin to a file, and fails the save for images named "broken".
func fakeDocker(t *testing.T) (log, stdin string) {
	t.Helper()
	bin := t.TempDir()
	log = filepath.Join(bin, "calls.log")
	stdin = filepath.Join(bin, "stdin")
	script := `echo "docker $*" >> ` + log + `
case "$1" in
  save) [ "$2" = broken ] && { echo "Error: no such image" >&2; exit 1; }; printf 'TARBYTES-%s' "$2";;
  exec) cat > ` + stdin + `; echo "unpacking";;
esac`
	_ = os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\n"+script+"\n"), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log, stdin
}

func TestImportImage_PipesSaveIntoCtrImport(t *testing.T) {
	log, stdin := fakeDocker(t)
	var out bytes.Buffer
	if err := ImportImage(context.Background(), "go-dark-rail-rancher", "local/til-order:1.0", &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "docker save local/til-order:1.0\n") ||
		!strings.Contains(string(calls), "docker exec -i go-dark-rail-rancher /usr/bin/ctr -a /run/k3s/containerd/containerd.sock -n k8s.io images import -\n") {
		t.Fatalf("calls:\n%s", calls)
	}
	if data, _ := os.ReadFile(stdin); string(data) != "TARBYTES-local/til-order:1.0" {
		t.Fatalf("ctr must read the saved image, got %q", data)
	}
	if !strings.Contains(out.String(), "imported into go-dark-rail-rancher") || !strings.Contains(out.String(), "MB streamed") {
		t.Fatalf("progress:\n%s", out.String())
	}
}

func TestImportImage_ReportsASaveFailure(t *testing.T) {
	fakeDocker(t)
	var out bytes.Buffer
	err := ImportImage(context.Background(), "c", "broken", &out)
	if err == nil || !strings.Contains(err.Error(), "docker save broken") || !strings.Contains(out.String(), "no such image") {
		t.Fatalf("expected the save error, got %v\n%s", err, out.String())
	}
}
