package networking

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	"ssh-bastion/config"
	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
)

func generateSessionID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func HandleConnection(tcpConn net.Conn, sshConfig *ssh.ServerConfig) {
	sid := generateSessionID()
	defer tcpConn.Close()

	log.Printf("[%s] New connection from %s", sid, tcpConn.RemoteAddr())

	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, sshConfig)
	if err != nil {
		log.Printf("[%s] SSH handshake failed from %s: %v", sid, tcpConn.RemoteAddr(), err)
		return
	}
	defer sshConn.Close()
	log.Printf("[%s] SSH connection established from %s (user=%s)", sid, sshConn.RemoteAddr(), sshConn.User())

	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		log.Printf("[%s] New channel type=%s", sid, newChan.ChannelType())
		switch newChan.ChannelType() {
		case "session":
			go handleSession(sid, sshConn, newChan)
		default:
			log.Printf("[%s] Rejecting unknown channel type: %s", sid, newChan.ChannelType())
			newChan.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChan.ChannelType()))
		}
	}
	log.Printf("[%s] Connection closed", sid)
}

func handleSession(sid string, sshConn *ssh.ServerConn, newChan ssh.NewChannel) {
	channel, requests, err := newChan.Accept()
	if err != nil {
		log.Printf("[%s] Failed to accept session channel: %v", sid, err)
		return
	}
	defer channel.Close()

	var ptyWidth, ptyHeight int
	agentForwarding := false
	shellReady := make(chan bool, 1)

	go func() {
		for req := range requests {
			log.Printf("[%s] Session request type=%s wantReply=%v", sid, req.Type, req.WantReply)
			switch req.Type {
			case "pty-req":
				ptyWidth, ptyHeight = ParsePtyRequest(req.Payload)
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "window-change":
				ptyWidth, ptyHeight = ParseWindowChange(req.Payload)
			case "auth-agent-req@openssh.com":
				log.Printf("[%s] Agent forwarding requested", sid)
				agentForwarding = true
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "shell":
				if req.WantReply {
					req.Reply(true, nil)
				}
				shellReady <- true
			case "env":
				if req.WantReply {
					req.Reply(true, nil)
				}
			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()

	// Wait for shell
	<-shellReady

	if !agentForwarding {
		log.Printf("[%s] No agent forwarding, rejecting session", sid)
		fmt.Fprintf(channel, "\r\nError: SSH agent forwarding not available.\r\n")
		fmt.Fprintf(channel, "Please reconnect with: ssh -A -p 2222 user@host\r\n")
		return
	}

	// Single reader goroutine for the entire session lifetime.
	// Both the menu and target connections consume from this channel,
	// preventing orphaned goroutines from stealing keypresses.
	inputCh := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := channel.Read(buf)
			if err != nil {
				close(inputCh)
				return
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			inputCh <- data
		}
	}()

	// Look up the bastion user from permissions
	bastionUsername := sshConn.Permissions.Extensions["bastion_user"]
	var bastionUser *models.User
	for i := range config.Cfg.Users {
		if config.Cfg.Users[i].Name == bastionUsername {
			bastionUser = &config.Cfg.Users[i]
			break
		}
	}
	if bastionUser == nil {
		log.Printf("[%s] Unknown bastion user", sid)
		fmt.Fprintf(channel, "\r\nError: unknown bastion user.\r\n")
		return
	}

	log.Printf("[%s] Bastion user: %s", sid, bastionUser.Name)

	// Menu loop — returns to menu when target connection ends
	for {
		target, connectUser, ok := ShowMenu(channel, inputCh, bastionUser)
		if !ok {
			log.Printf("[%s] User quit menu", sid)
			return
		}

		log.Printf("[%s] Connecting to target %s as %s", sid, target.Name, connectUser)
		err := ConnectToTarget(sid, sshConn, channel, inputCh, target, connectUser, ptyWidth, ptyHeight)
		if err != nil {
			log.Printf("[%s] Target session error: %v", sid, err)
			fmt.Fprintf(channel, "\r\n\033[1;31mSession ended: %v\033[0m\r\n", err)
		} else {
			log.Printf("[%s] Target session to %s closed cleanly", sid, target.Name)
			fmt.Fprintf(channel, "\r\n\033[1;33mConnection to %s closed.\033[0m\r\n", target.Name)
		}
		fmt.Fprintf(channel, "\r\nReturning to menu...\r\n\r\n")
	}
}
