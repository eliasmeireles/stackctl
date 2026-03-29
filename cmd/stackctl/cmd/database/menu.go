package database

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	dbcreate "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/create"
	dbdelete "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/delete"
	dblist "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/list"
	dbtest "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/test"
	vaultclient "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/generator"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var Menu list.Item

type dbEntry struct {
	display string
	cliType string
}

var dbEntries = []dbEntry{
	{"PostgreSQL", "postgres"},
	{"MySQL", "mysql"},
	{"MongoDB", "mongodb"},
}

func init() {
	initMenu()
}

func initMenu() {
	var dbTypeMenus []list.Item

	for _, entry := range dbEntries {
		e := entry
		items := []list.Item{
			vaultLoginSubMenu(
				"List",
				"List databases and users",
				e.cliType,
				nil,
				dbListAction(e.cliType),
			),
			ui.CreateSubMenu(
				"Create User",
				"Create a new database user",
				[]list.Item{
					ui.CreateSubMenu(
						"Auto-generate password",
						"Create user with a generated password",
						[]list.Item{
							vaultLoginSubMenu(
								"Select database from existing",
								"List and select an existing database",
								e.cliType,
								[]string{
									"New Username",
									"Password size (default: 16) (optional)",
									"Permissions (read / read-write, default: read-write) (optional)",
									"Vault Path to store credentials (e.g. secret/databases/" + e.cliType + "/myuser) (optional)",
								},
								dbCreateUserAutoNoDbAction(e.cliType),
							),
							vaultLoginSubMenu(
								"Enter database name",
								"Type the database name directly",
								e.cliType,
								[]string{
									"New Username",
									"Password size (default: 16) (optional)",
									"Database Name",
									"Permissions (read / read-write, default: read-write) (optional)",
									"Vault Path to store credentials (e.g. secret/databases/" + e.cliType + "/myuser) (optional)",
								},
								dbCreateUserAutoAction(e.cliType),
							),
						},
						"Choose the target database for the new user.",
					),
					ui.CreateSubMenu(
						"Enter password manually",
						"Create user with a custom password",
						[]list.Item{
							vaultLoginSubMenu(
								"Select database from existing",
								"List and select an existing database",
								e.cliType,
								[]string{
									"New Username",
									"New Password",
									"Permissions (read / read-write, default: read-write) (optional)",
									"Vault Path to store credentials (e.g. secret/databases/" + e.cliType + "/myuser) (optional)",
								},
								dbCreateUserNoDbAction(e.cliType),
							),
							vaultLoginSubMenu(
								"Enter database name",
								"Type the database name directly",
								e.cliType,
								[]string{
									"New Username",
									"New Password",
									"Database Name",
									"Permissions (read / read-write, default: read-write) (optional)",
									"Vault Path to store credentials (e.g. secret/databases/" + e.cliType + "/myuser) (optional)",
								},
								dbCreateUserAction(e.cliType),
							),
						},
						"Choose the target database for the new user.",
					),
				},
				"Choose how to set the password for the new user.",
			),
			vaultLoginSubMenu(
				"Delete User",
				"Delete an existing database user",
				e.cliType,
				nil,
				dbDeleteUserAction(e.cliType),
			),
			ui.CreateMultiPromptItemWithArgs(
				"Test Connection",
				"Test a user connection to "+e.display,
				[]string{
					"Host",
					"Username",
					"Password",
					"Database Name",
				},
				dbTestUserAction(e.cliType),
			),
		}

		dbTypeMenus = append(dbTypeMenus,
			ui.CreateSubMenu(e.display, "Manage "+e.display+" databases", items, "Select an operation to perform on "+e.display+"."),
		)
	}

	Menu = ui.CreateSubMenu("Database", "Manage databases (PostgreSQL, MySQL, MongoDB)", dbTypeMenus, "Select a database engine to manage.")
}

// vaultLoginSubMenu wraps an operation in a sub-menu offering both vault-path
// browsing and manual path entry.
func vaultLoginSubMenu(title, desc, dbType string, extraPrompts []string, action func(args []string) tea.Cmd) list.Item {
	vaultPrompt := "Vault Login Path (e.g. secret/databases/" + dbType + "/admin)"
	allManualPrompts := append([]string{vaultPrompt}, extraPrompts...)

	const vaultNote = "Select the Vault path that holds the admin credentials to authenticate with the database."
	return ui.CreateSubMenu(title, desc, []list.Item{
		vaultclient.BrowseVaultLoginItem(
			"Browse Vault (admin credentials)",
			"Navigate Vault to select the admin database credentials path",
			extraPrompts,
			action,
		),
		ui.CreateMultiPromptItemWithArgs(
			"Type admin credentials path",
			"Enter the Vault path where admin database credentials are stored",
			allManualPrompts,
			action,
		),
	}, vaultNote)
}

func dbListAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmdArgs := buildVaultLoginArgs(args, 0)
			cmdArgs = append(cmdArgs, "--dbs", "--users")

			cmd := ui.SilenceInteractive(dblist.NewListCommand(dbType))
			cmd.SetArgs(cmdArgs)
			return cmd.Execute()
		}
	}
}

// dbCreateUserAutoAction handles the "Auto-generate password" path.
// args: [vaultLogin, username, passwordSize, database, privileges, vaultPath]
// dbCreateUserAutoNoDbAction: auto-generate password, database selected post-TUI.
// args: [vaultLogin, username, passwordSize, privileges, vaultPath]
func dbCreateUserAutoNoDbAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmdArgs := buildVaultLoginArgs(args, 0)
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			sizeArg := strings.TrimSpace(argAt(args, 2))
			if sizeArg != "" {
				cmdArgs = append(cmdArgs, "--password", "auto:"+sizeArg)
			}
			// no --database: runCreateUser will list existing ones interactively
			if argAt(args, 3) != "" {
				cmdArgs = append(cmdArgs, "--privileges", normalizePrivileges(args[3]))
			}
			if argAt(args, 4) != "" {
				cmdArgs = append(cmdArgs, "--vault-path", args[4])
			}

			cmd := ui.SilenceInteractive(dbcreate.NewCreateCommand(dbType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

// dbCreateUserAutoAction: auto-generate password, database name typed manually.
// args: [vaultLogin, username, passwordSize, database, privileges, vaultPath]
func dbCreateUserAutoAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmdArgs := buildVaultLoginArgs(args, 0)
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			sizeArg := strings.TrimSpace(argAt(args, 2))
			if sizeArg != "" {
				cmdArgs = append(cmdArgs, "--password", "auto:"+sizeArg)
			}
			if argAt(args, 3) != "" {
				cmdArgs = append(cmdArgs, "--database", args[3])
			}
			if argAt(args, 4) != "" {
				cmdArgs = append(cmdArgs, "--privileges", normalizePrivileges(args[4]))
			}
			if argAt(args, 5) != "" {
				cmdArgs = append(cmdArgs, "--vault-path", args[5])
			}

			cmd := ui.SilenceInteractive(dbcreate.NewCreateCommand(dbType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

// dbCreateUserNoDbAction: manual password, database selected post-TUI.
// args: [vaultLogin, username, password, privileges, vaultPath]
func dbCreateUserNoDbAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmdArgs := buildVaultLoginArgs(args, 0)
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			if pw := resolvePassword(argAt(args, 2)); pw != "" {
				cmdArgs = append(cmdArgs, "--password", pw)
			}
			// no --database: runCreateUser will list existing ones interactively
			if argAt(args, 3) != "" {
				cmdArgs = append(cmdArgs, "--privileges", normalizePrivileges(args[3]))
			}
			if argAt(args, 4) != "" {
				cmdArgs = append(cmdArgs, "--vault-path", args[4])
			}

			cmd := ui.SilenceInteractive(dbcreate.NewCreateCommand(dbType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

func dbCreateUserAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			// args: [vaultLogin, username, password, database, privileges, vaultPath]
			cmdArgs := buildVaultLoginArgs(args, 0)
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			if pw := resolvePassword(argAt(args, 2)); pw != "" {
				cmdArgs = append(cmdArgs, "--password", pw)
			}
			// database may be blank — create_user.go will list existing ones interactively
			if argAt(args, 3) != "" {
				cmdArgs = append(cmdArgs, "--database", args[3])
			}
			if argAt(args, 4) != "" {
				cmdArgs = append(cmdArgs, "--privileges", normalizePrivileges(args[4]))
			}
			if argAt(args, 5) != "" {
				cmdArgs = append(cmdArgs, "--vault-path", args[5])
			}

			cmd := ui.SilenceInteractive(dbcreate.NewCreateCommand(dbType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

func dbDeleteUserAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			// args: [vaultLogin]
			// No --username: the CLI will list existing users and prompt for selection.
			// No --force: the CLI will ask for irreversible-action confirmation.
			cmdArgs := buildVaultLoginArgs(args, 0)

			cmd := ui.SilenceInteractive(dbdelete.NewDeleteCommand(dbType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

func dbTestUserAction(dbType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			// args: [host, username, password, database]
			cmdArgs := []string{}
			if argAt(args, 0) != "" {
				cmdArgs = append(cmdArgs, "--host", args[0])
			}
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			if argAt(args, 2) != "" {
				cmdArgs = append(cmdArgs, "--password", args[2])
			}
			if argAt(args, 3) != "" {
				cmdArgs = append(cmdArgs, "--database", args[3])
			}

			cmd := ui.SilenceInteractive(dbtest.NewTestUserCommand(dbType))
			cmd.SetArgs(cmdArgs)
			return cmd.Execute()
		}
	}
}

// resolvePassword returns the password to use. If raw starts with "auto" it generates
// a random one (optionally sized via "auto:N"), prints it, and returns it.
func resolvePassword(raw string) string {
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "auto") {
		return raw
	}
	size := 20
	if strings.HasPrefix(lower, "auto:") {
		if n, err := strconv.Atoi(strings.TrimPrefix(lower, "auto:")); err == nil && n > 0 {
			size = n
		}
	}
	pw, err := generator.NewPasswordGenerator().GeneratePassword(size)
	if err != nil {
		fmt.Printf("⚠️  Could not generate password: %v — please enter it manually\n", err)
		return ""
	}
	fmt.Printf("🔑 Generated password: %s\n", pw)
	return pw
}

// normalizePrivileges maps user-friendly labels to the value the --privileges flag accepts.
// "read" → "read", "read-write" / "rw" / "write" → "read-write"; anything else passes through.
func normalizePrivileges(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "r", "read", "readonly", "read-only":
		return "read"
	case "rw", "write", "read-write", "readwrite", "read_write":
		return "read-write"
	}
	return s
}

// buildVaultLoginArgs returns the --vault-login flag if args[idx] is non-empty.
func buildVaultLoginArgs(args []string, idx int) []string {
	if argAt(args, idx) != "" && !strings.HasPrefix(args[idx], "--") {
		return []string{"--vault-login", args[idx]}
	}
	return []string{}
}

// argAt returns args[idx] or "" if out of bounds.
func argAt(args []string, idx int) string {
	if idx < len(args) {
		return strings.TrimSpace(args[idx])
	}
	return ""
}
