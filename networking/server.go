package networking

import (
	"fmt"
	"log"
	"net"

	"ssh-bastion/config"
	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
)

func HandleConnection(tcpConn net.Conn, config *ssh.ServerConfig) {
	defer tcpConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		log.Printf("SSH handshake failed from %s: %v", tcpConn.RemoteAddr(), err)
		return
	}
	defer sshConn.Close()
	log.Printf("SSH connection established from %s (%s)", sshConn.RemoteAddr(), sshConn.User())

	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		log.Printf("[new-channel] type=%s", newChan.ChannelType())
		switch newChan.ChannelType() {
		case "session":
			go handleSession(sshConn, newChan)
		default:
			log.Printf("[channel] Rejecting unknown channel type: %s", newChan.ChannelType())
			newChan.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChan.ChannelType()))
		}
	}
}

func handleSession(sshConn *ssh.ServerConn, newChan ssh.NewChannel) {
	channel, requests, err := newChan.Accept()
	if err != nil {
		log.Printf("Failed to accept session channel: %v", err)
		return
	}
	defer channel.Close()

	var ptyWidth, ptyHeight int
	agentForwarding := false
	shellReady := make(chan bool, 1)

	go func() {
		for req := range requests {
			log.Printf("[session-req] type=%s wantReply=%v", req.Type, req.WantReply)
			switch req.Type {
			case "pty-req":
				ptyWidth, ptyHeight = ParsePtyRequest(req.Payload)
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "window-change":
				ptyWidth, ptyHeight = ParseWindowChange(req.Payload)
			case "auth-agent-req@openssh.com":
				log.Printf("[session-req] Agent forwarding requested")
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
		fmt.Fprintf(channel, "\r\nError: unknown bastion user.\r\n")
		return
	}

	// Menu loop — returns to menu when target connection ends
	for {
		target, connectUser, ok := ShowMenu(channel, inputCh, bastionUser)
		if !ok {
			return
		}

		err := ConnectToTarget(sshConn, channel, inputCh, target, connectUser, ptyWidth, ptyHeight)
		if err != nil {
			fmt.Fprintf(channel, "\r\n\033[1;31mSession ended: %v\033[0m\r\n", err)
		} else {
			fmt.Fprintf(channel, "\r\n\033[1;33mConnection to %s closed.\033[0m\r\n", target.Name)
		}
		fmt.Fprintf(channel, "\r\nReturning to menu...\r\n\r\n")
	}
}
