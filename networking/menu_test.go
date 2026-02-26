package networking

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-bastion/config"
	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
)

// mockChannel implements ssh.Channel for testing ShowMenu output and behavior.
type mockChannel struct {
	writeBuffer bytes.Buffer
	readErr     error
}

func (m *mockChannel) Read(data []byte) (int, error)  { return 0, m.readErr }
func (m *mockChannel) Write(data []byte) (int, error) { return m.writeBuffer.Write(data) }
func (m *mockChannel) Close() error                   { return nil }
func (m *mockChannel) CloseWrite() error              { return nil }
func (m *mockChannel) Stderr() io.ReadWriter          { return &bytes.Buffer{} }
func (m *mockChannel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return false, nil
}

var _ ssh.Channel = (*mockChannel)(nil)

var testAdminUser = &models.User{
	Name:   "admin",
	Groups: []string{"admin"},
}

var testDevUser = &models.User{
	Name:   "dev",
	Groups: []string{"dev"},
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "menu-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	cfg := `
listen_addr: ":2222"
host_key_path: "host_key"
users:
  - name: "admin"
    key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC85vp8YRG/as/vh3Aoax3D1ObkqwGUKnVWofMdbfeN6 test@example.com"
    groups:
      - "admin"
  - name: "dev"
    key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJf5p5PGmOYhJQ8mczMb+8aPJGlOaRnsn+JAm+CUVQ3G dev@example.com"
    groups:
      - "dev"
targets:
  - name: "Web Server"
    host: "10.0.0.1"
    port: "22"
    allow_groups:
      - "admin"
  - name: "Database Server"
    host: "10.0.0.2"
    port: "22"
    allow_groups:
      - "admin"
      - "dev"
  - name: "App Server"
    host: "10.0.0.3"
    port: "22"
    allow_groups:
      - "admin"
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(cfg), 0644); err != nil {
		panic(err)
	}
	if err := config.Load(path); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// --- resolveTarget tests ---

func TestResolveTarget_ByIndex(t *testing.T) {
	targets := config.Cfg.Targets
	tests := []struct {
		input    string
		wantName string
		wantNil  bool
	}{
		{"1", targets[0].Name, false},
		{"2", targets[1].Name, false},
		{"3", targets[2].Name, false},
		{"0", "", true},
		{"-1", "", true},
		{"99", "", true},
	}
	for _, tt := range tests {
		t.Run("index_"+tt.input, func(t *testing.T) {
			got := resolveTarget(tt.input, targets)
			if tt.wantNil {
				if got != nil {
					t.Errorf("resolveTarget(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("resolveTarget(%q) = nil, want %q", tt.input, tt.wantName)
			}
			if got.Name != tt.wantName {
				t.Errorf("resolveTarget(%q).Name = %q, want %q", tt.input, got.Name, tt.wantName)
			}
		})
	}
}

func TestResolveTarget_ByName(t *testing.T) {
	targets := config.Cfg.Targets
	for _, tgt := range targets {
		t.Run("name_"+tgt.Name, func(t *testing.T) {
			got := resolveTarget(tgt.Name, targets)
			if got == nil || got.Name != tgt.Name {
				t.Errorf("resolveTarget(%q) did not match", tgt.Name)
			}
			got = resolveTarget(strings.ToUpper(tgt.Name), targets)
			if got == nil || got.Name != tgt.Name {
				t.Errorf("resolveTarget(%q) case-insensitive did not match", strings.ToUpper(tgt.Name))
			}
		})
	}
}

func TestResolveTarget_ByHost(t *testing.T) {
	targets := config.Cfg.Targets
	for _, tgt := range targets {
		t.Run("host_"+tgt.Host, func(t *testing.T) {
			got := resolveTarget(tgt.Host, targets)
			if got == nil || got.Host != tgt.Host {
				t.Errorf("resolveTarget(%q) did not match by host", tgt.Host)
			}
		})
	}
}

func TestResolveTarget_Invalid(t *testing.T) {
	targets := config.Cfg.Targets
	invalids := []string{"", "nonexistent", "999", "abc123"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			if got := resolveTarget(input, targets); got != nil {
				t.Errorf("resolveTarget(%q) = %+v, want nil", input, got)
			}
		})
	}
}

// --- ShowMenu tests ---

func TestShowMenu_Quit(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	target, _, ok := ShowMenu(ch, inputCh, testAdminUser)
	if ok {
		t.Error("ShowMenu returned ok=true for quit")
	}
	if target != nil {
		t.Error("ShowMenu returned non-nil target for quit")
	}

	output := ch.writeBuffer.String()
	if !strings.Contains(output, "Goodbye") {
		t.Error("output missing 'Goodbye' message")
	}
}

func TestShowMenu_CtrlC(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte{3} // Ctrl-C
	close(inputCh)

	target, _, ok := ShowMenu(ch, inputCh, testAdminUser)
	if ok {
		t.Error("ShowMenu returned ok=true for Ctrl-C")
	}
	if target != nil {
		t.Error("ShowMenu returned non-nil target for Ctrl-C")
	}
}

func TestShowMenu_ValidSelection(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 4)
	inputCh <- []byte("1")
	inputCh <- []byte("\r")
	inputCh <- []byte("\r") // accept default user
	close(inputCh)

	target, connectUser, ok := ShowMenu(ch, inputCh, testAdminUser)
	if !ok {
		t.Fatal("ShowMenu returned ok=false for valid selection")
	}
	if target == nil {
		t.Fatal("ShowMenu returned nil target for valid selection")
	}
	if target.Name != config.Cfg.Targets[0].Name {
		t.Errorf("selected target = %q, want %q", target.Name, config.Cfg.Targets[0].Name)
	}
	if connectUser != "admin" {
		t.Errorf("connectUser = %q, want %q", connectUser, "admin")
	}
}

func TestShowMenu_InvalidThenValid(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 6)
	inputCh <- []byte("9")
	inputCh <- []byte("\r") // invalid selection
	inputCh <- []byte("1")
	inputCh <- []byte("\r") // valid selection
	inputCh <- []byte("\r") // accept default user
	close(inputCh)

	target, _, ok := ShowMenu(ch, inputCh, testAdminUser)
	if !ok {
		t.Fatal("ShowMenu returned ok=false")
	}
	if target == nil || target.Name != config.Cfg.Targets[0].Name {
		t.Errorf("expected first target after retry, got %+v", target)
	}

	output := ch.writeBuffer.String()
	if !strings.Contains(output, "Invalid selection") {
		t.Error("output missing 'Invalid selection' message")
	}
}

func TestShowMenu_ChannelClosed(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte)
	close(inputCh) // immediate close

	target, _, ok := ShowMenu(ch, inputCh, testAdminUser)
	if ok {
		t.Error("ShowMenu returned ok=true for closed channel")
	}
	if target != nil {
		t.Error("ShowMenu returned non-nil target for closed channel")
	}
}

func TestShowMenu_RendersAllowedTargets(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	ShowMenu(ch, inputCh, testAdminUser)

	output := ch.writeBuffer.String()
	// Admin should see all 3 targets
	for _, tgt := range config.Cfg.Targets {
		if !strings.Contains(output, tgt.Name) {
			t.Errorf("menu output missing target name %q", tgt.Name)
		}
	}
	if !strings.Contains(output, "SSH Bastion Host Gateway") {
		t.Error("menu output missing header")
	}
}

func TestShowMenu_FiltersTargetsByGroup(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	// Dev user should only see "Database Server" (allow_groups includes "dev")
	ShowMenu(ch, inputCh, testDevUser)

	output := ch.writeBuffer.String()
	if !strings.Contains(output, "Database Server") {
		t.Error("dev user should see Database Server")
	}
	if strings.Contains(output, "Web Server") {
		t.Error("dev user should NOT see Web Server")
	}
	if strings.Contains(output, "App Server") {
		t.Error("dev user should NOT see App Server")
	}
}

func TestShowMenu_WelcomeMessage(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	ShowMenu(ch, inputCh, testAdminUser)

	output := ch.writeBuffer.String()
	if !strings.Contains(output, "admin") {
		t.Error("menu output missing user name in welcome")
	}
}

func TestShowMenu_CustomUsername(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 4)
	inputCh <- []byte("1")
	inputCh <- []byte("\r")
	inputCh <- []byte("root")
	inputCh <- []byte("\r")
	close(inputCh)

	_, connectUser, ok := ShowMenu(ch, inputCh, testAdminUser)
	if !ok {
		t.Fatal("ShowMenu returned ok=false")
	}
	if connectUser != "root" {
		t.Errorf("connectUser = %q, want %q", connectUser, "root")
	}
}

func TestShowMenu_Backspace(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 6)
	inputCh <- []byte("9")  // type '9'
	inputCh <- []byte{127}  // backspace
	inputCh <- []byte("1")  // type '1'
	inputCh <- []byte("\r") // enter
	inputCh <- []byte("\r") // accept default user
	close(inputCh)

	target, _, ok := ShowMenu(ch, inputCh, testAdminUser)
	if !ok {
		t.Fatal("ShowMenu returned ok=false")
	}
	if target == nil || target.Name != config.Cfg.Targets[0].Name {
		t.Errorf("expected first target after backspace correction, got %+v", target)
	}
}

// --- Admin custom host tests ---

func TestShowMenu_AdminSeesCustomHostOption(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	ShowMenu(ch, inputCh, testAdminUser)

	output := ch.writeBuffer.String()
	if !strings.Contains(output, "Custom host") {
		t.Error("admin user should see 'Custom host' option")
	}
}

func TestShowMenu_NonAdminNoCustomHostOption(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	ShowMenu(ch, inputCh, testDevUser)

	output := ch.writeBuffer.String()
	if strings.Contains(output, "Custom host") {
		t.Error("non-admin user should NOT see 'Custom host' option")
	}
}

func TestShowMenu_CustomHostWithPort(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 4)
	inputCh <- []byte("c")
	inputCh <- []byte("example.com:2222\r")
	inputCh <- []byte("root\r")
	close(inputCh)

	target, connectUser, ok := ShowMenu(ch, inputCh, testAdminUser)
	if !ok {
		t.Fatal("ShowMenu returned ok=false for custom host")
	}
	if target == nil {
		t.Fatal("ShowMenu returned nil target for custom host")
	}
	if target.Host != "example.com" {
		t.Errorf("target.Host = %q, want %q", target.Host, "example.com")
	}
	if target.Port != "2222" {
		t.Errorf("target.Port = %q, want %q", target.Port, "2222")
	}
	if connectUser != "root" {
		t.Errorf("connectUser = %q, want %q", connectUser, "root")
	}
}

func TestShowMenu_CustomHostDefaultPort(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 4)
	inputCh <- []byte("c")
	inputCh <- []byte("example.com\r")
	inputCh <- []byte("\r") // accept default username
	close(inputCh)

	target, connectUser, ok := ShowMenu(ch, inputCh, testAdminUser)
	if !ok {
		t.Fatal("ShowMenu returned ok=false for custom host")
	}
	if target == nil {
		t.Fatal("ShowMenu returned nil target for custom host")
	}
	if target.Host != "example.com" {
		t.Errorf("target.Host = %q, want %q", target.Host, "example.com")
	}
	if target.Port != "22" {
		t.Errorf("target.Port = %q, want %q", target.Port, "22")
	}
	if connectUser != "admin" {
		t.Errorf("connectUser = %q, want default %q", connectUser, "admin")
	}
}

func TestShowMenu_CustomHostEmptyAddr(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 3)
	inputCh <- []byte("c")
	inputCh <- []byte("\r") // empty address
	close(inputCh)

	_, _, ok := ShowMenu(ch, inputCh, testAdminUser)
	if ok {
		t.Error("ShowMenu should return ok=false for empty custom host address")
	}

	output := ch.writeBuffer.String()
	if !strings.Contains(output, "No address entered") {
		t.Error("output missing 'No address entered' message")
	}
}

func TestShowMenu_NonAdminCKeyIgnored(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 2)
	inputCh <- []byte("c") // should be ignored for non-admin
	inputCh <- []byte("q")
	close(inputCh)

	_, _, ok := ShowMenu(ch, inputCh, testDevUser)
	if ok {
		t.Error("ShowMenu should return ok=false when non-admin presses c then q")
	}
}
