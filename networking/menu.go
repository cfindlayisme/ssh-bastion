package networking

import (
	"fmt"
	"strconv"
	"strings"

	"ssh-bastion/config"
	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
)

func ShowMenu(channel ssh.Channel, inputCh <-chan []byte, user *models.User) (*models.Target, string, bool) {
	// Filter targets by user's groups
	var targets []models.Target
	for _, t := range config.Cfg.Targets {
		if t.AllowedByGroups(user.Groups) {
			targets = append(targets, t)
		}
	}

	if len(targets) == 0 {
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
