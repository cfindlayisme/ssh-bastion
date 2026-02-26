package models

import "net"

type Target struct {
	Name        string   `yaml:"name"`
	Host        string   `yaml:"host"`
	Port        string   `yaml:"port"`
	AllowGroups []string `yaml:"allow_groups"`
}

func (t Target) Addr() string {
	return net.JoinHostPort(t.Host, t.Port)
}

func (t Target) AllowedByGroups(userGroups []string) bool {
	for _, ug := range userGroups {
		for _, ag := range t.AllowGroups {
			if ug == ag {
				return true
			}
		}
	}
	return false
}
