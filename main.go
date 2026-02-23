package main

import (
	"bytes"
	"fmt"
	"log"
	"net"

	"ssh-bastion/config"
	"ssh-bastion/crypto"
	"ssh-bastion/networking"

	"golang.org/x/crypto/ssh"
)

func main() {
	if err := config.Load(config.DefaultConfigPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	authorizedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(config.Cfg.AuthorizedKey))
	if err != nil {
		log.Fatalf("Failed to parse authorized key: %v", err)
	}

	hostKey, err := crypto.LoadOrGenerateHostKey(config.Cfg.HostKeyPath)
	if err != nil {
		log.Fatalf("Failed to load/generate host key: %v", err)
	}

	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), authorizedKey.Marshal()) {
				log.Printf("Accepted public key for %s from %s", conn.User(), conn.RemoteAddr())
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unknown public key for %s", conn.User())
		},
	}
	sshConfig.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", config.Cfg.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", config.Cfg.ListenAddr, err)
	}
	log.Printf("SSH bastion listening on %s", config.Cfg.ListenAddr)

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go networking.HandleConnection(tcpConn, sshConfig)
	}
}
