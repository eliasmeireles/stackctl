package delete

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// multiSelectFromList is a variable so tests can bypass the checkbox selector.
var multiSelectFromList = ui.MultiSelectFromList

// isInteractiveTerminal is a variable so tests can force either mode.
var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// deleteUsersBatch resolves the usernames, asks one confirmation for the
// whole batch and deletes every user, printing a per-user result.
func deleteUsersBatch(ctx context.Context, admin userAdmin, flags *DeleteUserFlags) error {
	names, err := resolveUsernames(ctx, admin, flags)
	if err != nil {
		return err
	}
	if !flags.Force {
		if err := confirmDeletion("user", names); err != nil {
			return err
		}
	}

	items, failed := deleteUsers(ctx, admin, names)
	output.PrintList("Deleted users", []string{"USERNAME", "STATUS"}, items)
	if failed > 0 {
		return fmt.Errorf("failed to delete %d of %d users", failed, len(names))
	}
	return nil
}

// resolveUsernames returns the explicit usernames, or lets the user check
// them from the broker's users (excluding the admin) in a terminal.
func resolveUsernames(ctx context.Context, admin userAdmin, flags *DeleteUserFlags) ([]string, error) {
	if names := normalizeNames(flags.Usernames); len(names) > 0 {
		return names, nil
	}
	if !isInteractiveTerminal() {
		return nil, fmt.Errorf("username is required")
	}
	users, err := admin.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	candidates := filterAdminUser(users, flags.AdminUser)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no users found to delete")
	}
	return multiSelectFromList("Select users to delete:", candidates)
}

// deleteUsers deletes each user, continuing on failure, and returns the
// per-user results and the number of failures.
func deleteUsers(ctx context.Context, admin userAdmin, names []string) ([]output.ListItem, int) {
	items := make([]output.ListItem, 0, len(names))
	failed := 0
	for _, name := range names {
		status := "deleted"
		if err := deleteUser(ctx, admin, name); err != nil {
			status = "failed: " + err.Error()
			failed++
		}
		items = append(items, output.NewItem("username", name, "status", status))
	}
	return items, failed
}

func deleteUser(ctx context.Context, admin userAdmin, name string) error {
	exists, err := admin.UserExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user does not exist")
	}
	output.Progress("🗑️  Deleting user '%s'...", name)
	return admin.RemoveUser(ctx, name)
}

// normalizeNames splits comma-separated values, trims blanks and removes
// duplicates while keeping the original order.
func normalizeNames(raw []string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			name := strings.TrimSpace(part)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// filterAdminUser returns user names excluding the current admin (cannot delete self).
func filterAdminUser(users []entity.UserInfo, adminUser string) []string {
	var names []string
	for _, u := range users {
		if u.Name != adminUser {
			names = append(names, u.Name)
		}
	}
	return names
}
