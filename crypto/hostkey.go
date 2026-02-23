package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func LoadOrGenerateHostKey(path string) (ssh.Signer, error) {
	keyData, err := os.ReadFile(path)
	if err == nil {
		signer, err := ssh.ParsePrivateKey(keyData)
		if err == nil {
			log.Printf("Loaded host key from %s", path)
			return signer, nil
		}
	}

	log.Printf("Generating new ED25519 host key...")
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})

	if err := os.WriteFile(path, pemBlock, 0600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	log.Printf("Host key saved to %s", path)

	signer, err := ssh.ParsePrivateKey(pemBlock)
	if err != nil {
		return nil, fmt.Errorf("parse generated key: %w", err)
	}

	return signer, nil
}
