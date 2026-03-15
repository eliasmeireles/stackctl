package entity

import "strings"

type UserInfo struct {
	Name        string
	Permissions []string
}

func (u UserInfo) PermissionsString() string {
	if len(u.Permissions) == 0 {
		return "(none)"
	}
	return strings.Join(u.Permissions, ", ")
}
