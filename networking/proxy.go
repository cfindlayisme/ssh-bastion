package networking

import (
	"fmt"
	"io"
	"log"
	"sync"

	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func ConnectToTarget(sshConn *ssh.ServerConn, channel ssh.Channel, inputCh <-chan []byte, target *models.Target, connectUser string, ptyWidth, ptyHeight int) error {
	// Open agent channel TO the client (server initiates this per SSH protocol)
	log.Printf("[agent] Opening auth-agent channel to client")
	agentChan, agentReqs, err := sshConn.OpenChannel("auth-agent@openssh.com", nil)
	if err != nil {
		return fmt.Errorf("failed to open agent channel: %v", err)
	}
	defer agentChan.Close()
	go ssh.DiscardRequests(agentReqs)

	ac := agent.NewClient(agentChan)
	log.Printf("[agent] Agent client ready")

	fmt.Fprintf(channel, "\r\nConnecting to %s...\r\n", target.Name)

	targetConfig := &ssh.ClientConfig{
		User: connectUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ac.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: use known_hosts
	}

	targetConn, err := ssh.Dial("tcp", target.Addr(), targetConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", target.Name, err)
	}
	defer targetConn.Close()

	targetSession, err := targetConn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to open session on %s: %v", target.Name, err)
	}
	defer targetSession.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := targetSession.RequestPty("xterm-256color", ptyHeight, ptyWidth, modes); err != nil {
		return fmt.Errorf("failed to request PTY on %s: %v", target.Name, err)
	}

	targetStdin, err := targetSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stdin: %v", err)
	}
	targetStdout, err := targetSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stdout: %v", err)
	}
	targetStderr, err := targetSession.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr: %v", err)
	}

	if err := targetSession.Shell(); err != nil {
		return fmt.Errorf("failed to start shell on %s: %v", target.Name, err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target (reads from shared inputCh, stops when done is closed)
	go func() {
		for {
			select {
			case data, ok := <-inputCh:
				if !ok {
					targetStdin.Close()
					return
				}
				targetStdin.Write(data)
			case <-done:
				return
			}
		}
	}()
	// Target stdout -> Client
	go func() {
		defer wg.Done()
		io.Copy(channel, targetStdout)
	}()
	// Target stderr -> Client
	go func() {
		defer wg.Done()
		io.Copy(channel.Stderr(), targetStderr)
	}()

	err = targetSession.Wait()
	close(done)
	wg.Wait()
	log.Printf("Session to %s (%s) ended", target.Name, target.Addr())
	return err
}
