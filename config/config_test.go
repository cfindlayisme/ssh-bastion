package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validConfig = `
listen_addr: ":2222"
host_key_path: "host_key"
authorized_key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC85vp8YRG/as/vh3Aoax3D1ObkqwGUKnVWofMdbfeN6 test@example.com"
targets:
  - name: "Web Server"
    host: "10.0.0.1"
    port: "22"
    user: "deploy"
  - name: "Database Server"
    host: "10.0.0.2"
    port: "22"
    user: "admin"
`

func TestLoad_Valid(t *testing.T) {
	path := writeTestConfig(t, validConfig)
	if err := Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
}

func TestLoad_ListenAddrFormat(t *testing.T) {
	path := writeTestConfig(t, validConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(Cfg.ListenAddr, ":") {
		t.Errorf("ListenAddr %q missing port separator", Cfg.ListenAddr)
	}
}

func TestLoad_AuthorizedKeyParseable(t *testing.T) {
	path := writeTestConfig(t, validConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(Cfg.AuthorizedKey))
	if err != nil {
		t.Fatalf("AuthorizedKey is not a valid SSH public key: %v", err)
	}
}

func TestLoad_TargetsNotEmpty(t *testing.T) {
	path := writeTestConfig(t, validConfig)
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if len(Cfg.Targets) == 0 {
		t.Fatal("Targets list is empty")
	}
}

func TestLoad_TargetsHaveRequiredFields(t *testing.T) {
	path := writeTestConfig(t, validConfig)
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
	cfg := `
listen_addr: ":2222"
targets:
  - name: "test"
    host: "localhost"
    port: "22"
    user: "user"
`
	path := writeTestConfig(t, cfg)
	err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing authorized_key")
	}
}

func TestLoad_NoTargets(t *testing.T) {
	cfg := `
authorized_key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC85vp8YRG/as/vh3Aoax3D1ObkqwGUKnVWofMdbfeN6 test@example.com"
targets: []
`
	path := writeTestConfig(t, cfg)
	err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg := `
authorized_key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC85vp8YRG/as/vh3Aoax3D1ObkqwGUKnVWofMdbfeN6 test@example.com"
targets:
  - name: "test"
    host: "localhost"
    port: "22"
    user: "user"
`
	path := writeTestConfig(t, cfg)
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
