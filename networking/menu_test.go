package networking

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-bastion/config"

	"golang.org/x/crypto/ssh"
)

// mockChannel implements ssh.Channel for testing ShowMenu output and behavior.
type mockChannel struct {
	writeBuffer bytes.Buffer
	readErr     error
}

func (m *mockChannel) Read(data []byte) (int, error)  { return 0, m.readErr }
func (m *mockChannel) Write(data []byte) (int, error)  { return m.writeBuffer.Write(data) }
func (m *mockChannel) Close() error                     { return nil }
func (m *mockChannel) CloseWrite() error                { return nil }
func (m *mockChannel) Stderr() io.ReadWriter            { return &bytes.Buffer{} }
func (m *mockChannel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return false, nil
}

var _ ssh.Channel = (*mockChannel)(nil)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "menu-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	cfg := `
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
  - name: "App Server"
    host: "10.0.0.3"
    port: "22"
    user: "app"
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
	tests := []struct {
		input    string
		wantName string
		wantNil  bool
	}{
		{"1", config.Cfg.Targets[0].Name, false},
		{"2", config.Cfg.Targets[1].Name, false},
		{"3", config.Cfg.Targets[2].Name, false},
		{"0", "", true},
		{"-1", "", true},
		{"99", "", true},
	}
	for _, tt := range tests {
		t.Run("index_"+tt.input, func(t *testing.T) {
			got := resolveTarget(tt.input)
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
	for _, tgt := range config.Cfg.Targets {
		t.Run("name_"+tgt.Name, func(t *testing.T) {
			// Exact name
			got := resolveTarget(tgt.Name)
			if got == nil || got.Name != tgt.Name {
				t.Errorf("resolveTarget(%q) did not match", tgt.Name)
			}
			// Case-insensitive
			got = resolveTarget(strings.ToUpper(tgt.Name))
			if got == nil || got.Name != tgt.Name {
				t.Errorf("resolveTarget(%q) case-insensitive did not match", strings.ToUpper(tgt.Name))
			}
		})
	}
}

func TestResolveTarget_ByHost(t *testing.T) {
	for _, tgt := range config.Cfg.Targets {
		t.Run("host_"+tgt.Host, func(t *testing.T) {
			got := resolveTarget(tgt.Host)
			if got == nil || got.Host != tgt.Host {
				t.Errorf("resolveTarget(%q) did not match by host", tgt.Host)
			}
		})
	}
}

func TestResolveTarget_Invalid(t *testing.T) {
	invalids := []string{"", "nonexistent", "999", "abc123"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			if got := resolveTarget(input); got != nil {
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

	target, ok := ShowMenu(ch, inputCh)
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

	target, ok := ShowMenu(ch, inputCh)
	if ok {
		t.Error("ShowMenu returned ok=true for Ctrl-C")
	}
	if target != nil {
		t.Error("ShowMenu returned non-nil target for Ctrl-C")
	}
}

func TestShowMenu_ValidSelection(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 2)
	inputCh <- []byte("1")
	inputCh <- []byte("\r")
	close(inputCh)

	target, ok := ShowMenu(ch, inputCh)
	if !ok {
		t.Fatal("ShowMenu returned ok=false for valid selection")
	}
	if target == nil {
		t.Fatal("ShowMenu returned nil target for valid selection")
	}
	if target.Name != config.Cfg.Targets[0].Name {
		t.Errorf("selected target = %q, want %q", target.Name, config.Cfg.Targets[0].Name)
	}
}

func TestShowMenu_InvalidThenValid(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 4)
	inputCh <- []byte("9")
	inputCh <- []byte("\r") // invalid selection
	inputCh <- []byte("1")
	inputCh <- []byte("\r") // valid selection
	close(inputCh)

	target, ok := ShowMenu(ch, inputCh)
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

	target, ok := ShowMenu(ch, inputCh)
	if ok {
		t.Error("ShowMenu returned ok=true for closed channel")
	}
	if target != nil {
		t.Error("ShowMenu returned non-nil target for closed channel")
	}
}

func TestShowMenu_RendersAllTargets(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 1)
	inputCh <- []byte("q")
	close(inputCh)

	ShowMenu(ch, inputCh)

	output := ch.writeBuffer.String()
	for _, tgt := range config.Cfg.Targets {
		if !strings.Contains(output, tgt.Name) {
			t.Errorf("menu output missing target name %q", tgt.Name)
		}
	}
	if !strings.Contains(output, "SSH Bastion Host Gateway") {
		t.Error("menu output missing header")
	}
}

func TestShowMenu_Backspace(t *testing.T) {
	ch := &mockChannel{}
	inputCh := make(chan []byte, 5)
	inputCh <- []byte("9")       // type '9'
	inputCh <- []byte{127}       // backspace
	inputCh <- []byte("1")       // type '1'
	inputCh <- []byte("\r")      // enter
	close(inputCh)

	target, ok := ShowMenu(ch, inputCh)
	if !ok {
		t.Fatal("ShowMenu returned ok=false")
	}
	if target == nil || target.Name != config.Cfg.Targets[0].Name {
		t.Errorf("expected first target after backspace correction, got %+v", target)
	}
}
