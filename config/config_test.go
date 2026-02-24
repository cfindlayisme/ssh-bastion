package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"ssh-bastion/models/mocks"
)

func writeMockConfig(t *testing.T, data []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
}

func TestLoad_ListenAddrFormat(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(Cfg.ListenAddr, ":") {
		t.Errorf("ListenAddr %q missing port separator", Cfg.ListenAddr)
	}
}

func TestLoad_AuthorizedKeyParseable(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(Cfg.AuthorizedKey))
	if err != nil {
		t.Fatalf("AuthorizedKey is not a valid SSH public key: %v", err)
	}
}

func TestLoad_TargetsNotEmpty(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if len(Cfg.Targets) == 0 {
		t.Fatal("Targets list is empty")
	}
}

func TestLoad_TargetsHaveRequiredFields(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	for i, tgt := range Cfg.Targets {
		if tgt.Name == "" {
			t.Errorf("Targets[%d] has empty Name", i)
		}
		if tgt.Host == "" {
			t.Errorf("Targets[%d] (%s) has empty Host", i, tgt.Name)
		}
		if tgt.Port == "" {
			t.Errorf("Targets[%d] (%s) has empty Port", i, tgt.Name)
		}
		if tgt.User == "" {
			t.Errorf("Targets[%d] (%s) has empty User", i, tgt.Name)
		}
	}
}

func TestLoad_MissingFile(t *testing.T) {
	err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MissingAuthorizedKey(t *testing.T) {
	path := writeMockConfig(t, mocks.MissingAuthorizedKey)
	err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing authorized_key")
	}
}

func TestLoad_NoTargets(t *testing.T) {
	path := writeMockConfig(t, mocks.NoTargets)
	err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeMockConfig(t, mocks.DefaultsOnly)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if Cfg.ListenAddr != ":2222" {
		t.Errorf("default ListenAddr = %q, want %q", Cfg.ListenAddr, ":2222")
	}
	if Cfg.HostKeyPath != "host_key" {
		t.Errorf("default HostKeyPath = %q, want %q", Cfg.HostKeyPath, "host_key")
	}
}
