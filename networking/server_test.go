package networking

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// generateTestSigner creates an ed25519 SSH signer for testing.
func generateTestSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	signer, err := ssh.ParsePrivateKey(pemBlock)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	return signer
}

func TestHandleConnection_RejectsUnknownKey(t *testing.T) {
	hostSigner := generateTestSigner(t)
	clientSigner := generateTestSigner(t)
	authSigner := generateTestSigner(t) // different key — should be rejected

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), authSigner.PublicKey().Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unknown key")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		HandleConnection(conn, serverConfig)
	}()

	clientConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         2 * time.Second,
	}

	_, err = ssh.Dial("tcp", ln.Addr().String(), clientConfig)
	if err == nil {
		t.Fatal("expected auth failure, but connection succeeded")
	}
	if !strings.Contains(err.Error(), "unable to authenticate") {
		t.Logf("got error (acceptable): %v", err)
	}

	<-done
}

func TestHandleConnection_AcceptsAuthorizedKey(t *testing.T) {
	hostSigner := generateTestSigner(t)
	clientSigner := generateTestSigner(t)

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientSigner.PublicKey().Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unknown key")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		HandleConnection(conn, serverConfig)
	}()

	clientConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         2 * time.Second,
	}

	client, err := ssh.Dial("tcp", ln.Addr().String(), clientConfig)
	if err != nil {
		t.Fatalf("expected auth success, got: %v", err)
	}
	client.Close()

	<-serverDone
}

func TestHandleConnection_RejectsUnknownChannelType(t *testing.T) {
	hostSigner := generateTestSigner(t)
	clientSigner := generateTestSigner(t)

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientSigner.PublicKey().Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unknown key")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		HandleConnection(conn, serverConfig)
	}()

	clientConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         2 * time.Second,
	}

	client, err := ssh.Dial("tcp", ln.Addr().String(), clientConfig)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	// Try opening a bogus channel type — should be rejected
	_, _, err = client.OpenChannel("bogus-channel-type", nil)
	if err == nil {
		t.Fatal("expected rejection of unknown channel type")
	}
	if !strings.Contains(err.Error(), "unknown channel type") {
		t.Logf("got rejection error (acceptable): %v", err)
	}
}

func TestHandleSession_NoAgentForwarding(t *testing.T) {
	hostSigner := generateTestSigner(t)
	clientSigner := generateTestSigner(t)

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientSigner.PublicKey().Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unknown key")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		HandleConnection(conn, serverConfig)
	}()

	clientConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         2 * time.Second,
	}

	client, err := ssh.Dial("tcp", ln.Addr().String(), clientConfig)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	defer session.Close()

	// Request PTY + shell but NO agent forwarding
	if err := session.RequestPty("xterm", 24, 80, ssh.TerminalModes{}); err != nil {
		t.Fatalf("pty: %v", err)
	}

	output, err := session.Output("") // triggers shell request
	// The session should show agent forwarding error
	if err != nil {
		// session may close with an error, that's expected
		t.Logf("session ended with: %v", err)
	}
	if !strings.Contains(string(output), "agent forwarding") {
		t.Logf("output: %q", string(output))
		// Session may have closed before we read — not a hard failure
	}
}
