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

	type parsedUser struct {
		name      string
		publicKey ssh.PublicKey
	}
	var authorizedUsers []parsedUser
	for _, u := range config.Cfg.Users {
		pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(u.Key))
		if err != nil {
			log.Fatalf("Failed to parse key for user %s: %v", u.Name, err)
		}
		authorizedUsers = append(authorizedUsers, parsedUser{name: u.Name, publicKey: pub})
	}

	hostKey, err := crypto.LoadOrGenerateHostKey(config.Cfg.HostKeyPath)
	if err != nil {
		log.Fatalf("Failed to load/generate host key: %v", err)
	}

	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			for _, u := range authorizedUsers {
				if bytes.Equal(key.Marshal(), u.publicKey.Marshal()) {
					log.Printf("Accepted public key for bastion user %s from %s", u.name, conn.RemoteAddr())
					return &ssh.Permissions{
						Extensions: map[string]string{
							"bastion_user": u.name,
						},
					}, nil
				}
			}
			return nil, fmt.Errorf("unknown public key from %s", conn.RemoteAddr())
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
