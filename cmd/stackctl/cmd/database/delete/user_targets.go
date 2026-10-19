package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

// userTarget is a user to delete: name is what is shown and reported,
// username is what the client receives and db is where it lives (MongoDB).
type userTarget struct {
	name     string
	username string
	db       string
}

// isSelf reports whether the target is the admin user the command is
// connected as.
func (u userTarget) isSelf(adminUser, authDB string) bool {
	return u.username == adminUser && u.db == authDB
}

// resolveUserTarget normalizes a listed or typed name. MongoDB lists users
// as "name@database"; MySQL lists them as "name@host" while removal works on
// the bare name.
func resolveUserTarget(dbType, name, authDB string) userTarget {
	switch dbType {
	case "mongodb":
		username, db := splitUserAt(name, authDB)
		return userTarget{name: username + "@" + db, username: username, db: db}
	case "mysql":
		username, _ := splitUserAt(name, "")
		return userTarget{name: username, username: username, db: authDB}
	default:
		return userTarget{name: name, username: name, db: authDB}
	}
}

func resolveUserTargets(dbType string, names []string, authDB string) []userTarget {
	seen := make(map[string]bool)
	targets := make([]userTarget, 0, len(names))
	for _, n := range names {
		u := resolveUserTarget(dbType, n, authDB)
		if seen[u.name] {
			continue
		}
		seen[u.name] = true
		targets = append(targets, u)
	}
	return targets
}

// userCandidates returns the picker entries: every listed user except the
// connected admin (cannot delete self), without duplicates.
func userCandidates(dbType string, users []entity.UserInfo, adminUser, authDB string) []string {
	names := make([]string, 0, len(users))
	for _, u := range users {
		names = append(names, u.Name)
	}
	var candidates []string
	for _, u := range resolveUserTargets(dbType, names, authDB) {
		if !u.isSelf(adminUser, authDB) {
			candidates = append(candidates, u.name)
		}
	}
	return candidates
}

func userLabels(dbType string, targets []userTarget) []string {
	labels := make([]string, len(targets))
	for i, u := range targets {
		labels[i] = u.name
		if dbType == "mongodb" {
			labels[i] = fmt.Sprintf("%s (database: %s)", u.username, u.db)
		}
	}
	return labels
}

// splitUserAt splits "username@suffix" (MongoDB database or MySQL host) at
// the last "@". If there is no "@", returns (name, fallback).
func splitUserAt(name, fallback string) (username, suffix string) {
	if idx := strings.LastIndex(name, "@"); idx != -1 {
		return name[:idx], name[idx+1:]
	}
	return name, fallback
}

// userPool keeps one connection per database so a batch spanning several
// MongoDB databases connects to each only once.
type userPool struct {
	ctx    context.Context
	target serverTarget
	stores map[string]userStore
}

func newUserPool(ctx context.Context, t serverTarget) *userPool {
	return &userPool{ctx: ctx, target: t, stores: make(map[string]userStore)}
}

func (p *userPool) get(db string) (userStore, error) {
	if s, ok := p.stores[db]; ok {
		return s, nil
	}
	s, err := newUserStore(p.ctx, p.target, db)
	if err != nil {
		return nil, err
	}
	p.stores[db] = s
	return s, nil
}

func (p *userPool) remove(u userTarget) error {
	store, err := p.get(u.db)
	if err != nil {
		return err
	}
	return deleteOneUser(p.ctx, store, u)
}

func (p *userPool) close() {
	for _, s := range p.stores {
		_ = s.Close()
	}
}
