package networking

import (
	"fmt"
	"strconv"
	"strings"

	"ssh-bastion/config"
	"ssh-bastion/models"

	"golang.org/x/crypto/ssh"
)

func ShowMenu(channel ssh.Channel, inputCh <-chan []byte) (*models.Target, bool) {
	fmt.Fprintf(channel, "\033[2J\033[H")
	fmt.Fprintf(channel, "\033[1;36m╔══════════════════════════════════════╗\033[0m\r\n")
	fmt.Fprintf(channel, "\033[1;36m║       SSH Bastion Host Gateway       ║\033[0m\r\n")
	fmt.Fprintf(channel, "\033[1;36m╚══════════════════════════════════════╝\033[0m\r\n")
	fmt.Fprintf(channel, "\r\n")
	fmt.Fprintf(channel, "  Select a host to connect to:\r\n\r\n")

	for i, t := range config.Cfg.Targets {
		fmt.Fprintf(channel, "  \033[1;33m%d)\033[0m %s \033[2m(%s@%s)\033[0m\r\n", i+1, t.Name, t.User, t.Addr())
	}

	fmt.Fprintf(channel, "\r\n  \033[1;33mq)\033[0m Quit\r\n")
	fmt.Fprintf(channel, "\r\n\033[1m> \033[0m")

	var input strings.Builder

	for data := range inputCh {
		for _, b := range data {
			switch {
			case b == 'q' || b == 'Q':
				fmt.Fprintf(channel, "q\r\nGoodbye!\r\n")
				return nil, false
			case b == 3: // Ctrl-C
				fmt.Fprintf(channel, "\r\nGoodbye!\r\n")
				return nil, false
			case b == '\r' || b == '\n':
				choice := strings.TrimSpace(input.String())
				if choice == "" {
					continue
				}
				target := resolveTarget(choice)
				if target == nil {
					fmt.Fprintf(channel, "\r\n  \033[1;31mInvalid selection.\033[0m\r\n\033[1m> \033[0m")
					input.Reset()
					continue
				}
				return target, true
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
	return nil, false
}

func resolveTarget(input string) *models.Target {
	if idx, err := strconv.Atoi(input); err == nil {
		if idx >= 1 && idx <= len(config.Cfg.Targets) {
			return &config.Cfg.Targets[idx-1]
		}
	}
	lower := strings.ToLower(input)
	for i := range config.Cfg.Targets {
		if strings.ToLower(config.Cfg.Targets[i].Name) == lower {
			return &config.Cfg.Targets[i]
		}
		if strings.ToLower(config.Cfg.Targets[i].Host) == lower {
			return &config.Cfg.Targets[i]
		}
	}
	return nil
}
