package models

type User struct {
	Name   string   `yaml:"name"`
	Key    string   `yaml:"key"`
	Groups []string `yaml:"groups"`
}

func (u *User) InGroup(group string) bool {
	for _, g := range u.Groups {
		if g == group {
			return true
		}
	}
	return false
}
