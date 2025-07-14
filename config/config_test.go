package config

import (
	"os"
	"testing"
)

func TestLoadConfigValid(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "conf-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmp.Close()

	content := `{"processes":[{"name":"echo","command":["echo","hi"],"autorun":false,"cwd":""}]}`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmp.Close()

	conf, err := LoadConfig(tmp.Name())
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if len(conf.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(conf.Processes))
	}
	p := conf.Processes[0]
	if p.Name != "echo" {
		t.Errorf("unexpected process name: %q", p.Name)
	}
	if len(p.Command) != 2 || p.Command[0] != "echo" || p.Command[1] != "hi" {
		t.Errorf("unexpected command: %#v", p.Command)
	}
	if p.Autorun != false {
		t.Errorf("unexpected autorun: %v", p.Autorun)
	}
	if p.Cwd != "" {
		t.Errorf("unexpected cwd: %q", p.Cwd)
	}
}

func TestLoadConfigFileNotExist(t *testing.T) {
	_, err := LoadConfig("nonexistent_file.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
