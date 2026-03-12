package vault

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

const DefaultPasswordSize = 16

func (a *Applier) applyUsers(u *UsersConfig) error {
	if u.AuthMount == "" {
		return fmt.Errorf("users.auth_mount is required")
	}

	authMount := strings.TrimRight(u.AuthMount, "/")

	if len(u.Add) > 0 {
		log.Infof("Adding %d users", len(u.Add))
		for _, entry := range u.Add {
			userPath := fmt.Sprintf("auth/%s/users/%s", authMount, entry.Username)

			existing, _ := a.logical.Read(userPath)
			if existing != nil {
				log.Infof("⚠️  User [%q] already exists. Skipping...", entry.Username)
				continue
			}

			password, err := resolveUserPassword(entry)
			if err != nil {
				return fmt.Errorf("resolve password for user %q: %w", entry.Username, err)
			}

			data := buildUserData(entry, password)
			if err := a.logical.Write(userPath, data); err != nil {
				return fmt.Errorf("create user %q: %w", entry.Username, err)
			}

			if entry.AutoGeneratePass {
				log.Infof("✅ User [%q] created with auto-generated password: %s", entry.Username, password)
			}
		}
	}

	if len(u.Update) > 0 {
		log.Infof("Updating %d users", len(u.Update))
		for _, entry := range u.Update {
			userPath := fmt.Sprintf("auth/%s/users/%s", authMount, entry.Username)

			password, err := resolveUserPassword(entry)
			if err != nil {
				return fmt.Errorf("resolve password for user %q: %w", entry.Username, err)
			}

			data := buildUserData(entry, password)
			if err := a.logical.Write(userPath, data); err != nil {
				return fmt.Errorf("update user %q: %w", entry.Username, err)
			}
		}
	}

	if len(u.Delete) > 0 {
		log.Infof("Deleting %d users", len(u.Delete))
		for _, entry := range u.Delete {
			userPath := fmt.Sprintf("auth/%s/users/%s", authMount, entry.Username)

			if err := a.logical.Delete(userPath); err != nil {
				return fmt.Errorf("delete user %q: %w", entry.Username, err)
			}
		}
	}

	return nil
}

func resolveUserPassword(entry UserEntry) (string, error) {
	if entry.AutoGeneratePass {
		size := entry.PasswordSize
		if size <= 0 {
			size = DefaultPasswordSize
		}
		b := make([]byte, size)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("generate random password: %w", err)
		}
		return hex.EncodeToString(b), nil
	}

	if entry.Password == "" {
		return "", fmt.Errorf("password is required when auto_generate_password is false")
	}

	return entry.Password, nil
}

func buildUserData(entry UserEntry, password string) map[string]interface{} {
	data := make(map[string]interface{})

	data["password"] = password

	if len(entry.Policies) > 0 {
		data["policies"] = strings.Join(entry.Policies, ",")
		data["token_policies"] = strings.Join(entry.Policies, ",")
	}

	return data
}
