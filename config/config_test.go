package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestLoad_UsersNotEmpty(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if len(Cfg.Users) == 0 {
		t.Fatal("Users list is empty")
	}
}

func TestLoad_UsersHaveRequiredFields(t *testing.T) {
	path := writeMockConfig(t, mocks.ValidConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	for i, u := range Cfg.Users {
		if u.Name == "" {
			t.Errorf("Users[%d] has empty Name", i)
		}
		if u.Key == "" {
			t.Errorf("Users[%d] (%s) has empty Key", i, u.Name)
		}
		if len(u.Groups) == 0 {
			t.Errorf("Users[%d] (%s) has no groups", i, u.Name)
		}
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
		if len(tgt.AllowGroups) == 0 {
			t.Errorf("Targets[%d] (%s) has no allow_groups", i, tgt.Name)
		}
	}
}

func TestLoad_MissingFile(t *testing.T) {
	err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MissingUsers(t *testing.T) {
	path := writeMockConfig(t, mocks.MissingUsers)
	err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing users")
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
