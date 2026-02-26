package networking

import (
	"fmt"
	"strconv"
	"strings"

	"ssh-bastion/config"
	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
)

const adminGroup = "admin"

func ShowMenu(channel ssh.Channel, inputCh <-chan []byte, user *models.User) (*models.Target, string, bool) {
	// Filter targets by user's groups
	var targets []models.Target
	for _, t := range config.Cfg.Targets {
		if t.AllowedByGroups(user.Groups) {
			targets = append(targets, t)
		}
	}

	isAdmin := user.InGroup(adminGroup)

	if len(targets) == 0 && !isAdmin {
		fmt.Fprintf(channel, "\r\n\033[1;31mNo targets available for your account.\033[0m\r\n")
		return nil, "", false
	}

	fmt.Fprintf(channel, "\033[2J\033[H")
	fmt.Fprintf(channel, "\033[1;36m╔══════════════════════════════════════╗\033[0m\r\n")
	fmt.Fprintf(channel, "\033[1;36m║       SSH Bastion Host Gateway       ║\033[0m\r\n")
	fmt.Fprintf(channel, "\033[1;36m╚══════════════════════════════════════╝\033[0m\r\n")
	fmt.Fprintf(channel, "\r\n")
	fmt.Fprintf(channel, "  Welcome, \033[1;32m%s\033[0m\r\n\r\n", user.Name)
	fmt.Fprintf(channel, "  Select a host to connect to:\r\n\r\n")

	for i, t := range targets {
		fmt.Fprintf(channel, "  \033[1;33m%d)\033[0m %s\r\n", i+1, t.Name)
	}

	if isAdmin {
		fmt.Fprintf(channel, "\r\n  \033[1;33mc)\033[0m Custom host\r\n")
	}

	fmt.Fprintf(channel, "\r\n  \033[1;33mq)\033[0m Quit\r\n")
	fmt.Fprintf(channel, "\r\n\033[1m> \033[0m")

	var input strings.Builder

	for data := range inputCh {
		for _, b := range data {
			switch {
			case b == 'q' || b == 'Q':
				fmt.Fprintf(channel, "q\r\nGoodbye!\r\n")
				return nil, "", false
			case b == 3: // Ctrl-C
				fmt.Fprintf(channel, "\r\nGoodbye!\r\n")
				return nil, "", false
			case (b == 'c' || b == 'C') && isAdmin:
				fmt.Fprintf(channel, "c\r\n")
				return promptCustomHost(channel, inputCh, user)
			case b == '\r' || b == '\n':
				choice := strings.TrimSpace(input.String())
				if choice == "" {
					continue
				}
				target := resolveTarget(choice, targets)
				if target == nil {
					fmt.Fprintf(channel, "\r\n  \033[1;31mInvalid selection.\033[0m\r\n\033[1m> \033[0m")
					input.Reset()
					continue
				}
				connectUser, ok := promptUser(channel, inputCh, target, user)
				if !ok {
					return nil, "", false
				}
				return target, connectUser, true
			case b >= '0' && b <= '9':
				input.WriteByte(b)
				fmt.Fprintf(channel, "%c", b)
			case b == 127 || b == 8: // Backspace
				s := input.String()
				if len(s) > 0 {
					input.Reset()
					input.WriteString(s[:len(s)-1])
					fmt.Fprintf(channel, "\b \b")
				}
			}
		}
	}
	return nil, "", false
}

func promptCustomHost(channel ssh.Channel, inputCh <-chan []byte, user *models.User) (*models.Target, string, bool) {
	// Prompt for host:port
	fmt.Fprintf(channel, "\r\n  Enter host:port: ")
	addr, ok := readLine(channel, inputCh)
	if !ok {
		return nil, "", false
	}
	addr = strings.TrimSpace(addr)
	if addr == "" {
		fmt.Fprintf(channel, "\r\n  \033[1;31mNo address entered.\033[0m\r\n")
		return nil, "", false
	}

	// Parse host and port, default to port 22 if not specified
	host, port := addr, "22"
	if i := strings.LastIndex(addr, ":"); i != -1 {
		host = addr[:i]
		port = addr[i+1:]
	}

	// Prompt for username
	fmt.Fprintf(channel, "  Enter username [\033[1;32m%s\033[0m]: ", user.Name)
	username, ok := readLine(channel, inputCh)
	if !ok {
		return nil, "", false
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = user.Name
	}

	target := &models.Target{
		Name: host,
		Host: host,
		Port: port,
	}
	return target, username, true
}

func readLine(channel ssh.Channel, inputCh <-chan []byte) (string, bool) {
	var input strings.Builder
	for data := range inputCh {
		for _, b := range data {
			switch {
			case b == 3: // Ctrl-C
				fmt.Fprintf(channel, "\r\n")
				return "", false
			case b == '\r' || b == '\n':
				fmt.Fprintf(channel, "\r\n")
				return input.String(), true
			case b == 127 || b == 8: // Backspace
				s := input.String()
				if len(s) > 0 {
					input.Reset()
					input.WriteString(s[:len(s)-1])
					fmt.Fprintf(channel, "\b \b")
				}
			default:
				if b >= 32 && b < 127 {
					input.WriteByte(b)
					fmt.Fprintf(channel, "%c", b)
				}
			}
		}
	}
	return "", false
}

func promptUser(channel ssh.Channel, inputCh <-chan []byte, target *models.Target, user *models.User) (string, bool) {
	fmt.Fprintf(channel, "\r\n\r\n  Connect to \033[1m%s\033[0m as \033[1;32m%s\033[0m? [Y/n/username]: ", target.Name, user.Name)

	var input strings.Builder

	for data := range inputCh {
		for _, b := range data {
			switch {
			case b == 3: // Ctrl-C
				fmt.Fprintf(channel, "\r\n")
				return "", false
			case b == '\r' || b == '\n':
				choice := strings.TrimSpace(input.String())
				if choice == "" || strings.EqualFold(choice, "y") {
					fmt.Fprintf(channel, "\r\n")
					return user.Name, true
				}
				if strings.EqualFold(choice, "n") {
					fmt.Fprintf(channel, "\r\n")
					return "", false
				}
				fmt.Fprintf(channel, "\r\n")
				return choice, true
			case b == 127 || b == 8: // Backspace
				s := input.String()
				if len(s) > 0 {
					input.Reset()
					input.WriteString(s[:len(s)-1])
					fmt.Fprintf(channel, "\b \b")
				}
			default:
				if b >= 32 && b < 127 {
					input.WriteByte(b)
					fmt.Fprintf(channel, "%c", b)
				}
			}
		}
	}
	return "", false
}

func resolveTarget(input string, targets []models.Target) *models.Target {
	if idx, err := strconv.Atoi(input); err == nil {
		if idx >= 1 && idx <= len(targets) {
			return &targets[idx-1]
		}
	}
	lower := strings.ToLower(input)
	for i := range targets {
		if strings.ToLower(targets[i].Name) == lower {
			return &targets[i]
		}
		if strings.ToLower(targets[i].Host) == lower {
			return &targets[i]
		}
	}
	return nil
}
