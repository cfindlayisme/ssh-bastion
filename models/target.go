package models

import "net"

type Target struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	User string `yaml:"user"`
}

func (t Target) Addr() string {
	return net.JoinHostPort(t.Host, t.Port)
}
