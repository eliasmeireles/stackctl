package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Command defines the interface for all commands.
type Command interface {
	// Category returns the command matching on execution flow. The matching is done using regex.
	//
	// Example:
	// 	- "Vault/Secrets/Delete" for each vault secret deletion command.
	//  - "K8s Config/Set Context" for each k8s context setting command.
	Category() string
	Execute(choice, args []string) bool
}

// Commands is a map of Category to Command.
type Commands map[string]Command

var (
	cmds = make(Commands)
)

func Cmd() *Commands {
	return &cmds
}

func Add(cmd Command) *Commands {
	return Cmd().Add(cmd)
}

// Add adds a command to the collection.
func (c *Commands) Add(cmd Command) *Commands {
	if cmd == nil {
		return c
	}

	category := (cmd).Category()
	// Replace whitespace with _ and convert to lowercase
	category = categorySanitize(category)
	(*c)[category] = cmd
	return c
}

func categorySanitize(category string) string {
	return category
}

// Get retrieves a command by its category using regex matching.
// It checks if the provided category starts with any of the registered command categories.
func (c *Commands) Get(category string) (Command, bool) {
	if category == "" {
		return nil, false
	}

	sanitizedCategory := categorySanitize(category)

	for k, v := range *c {
		// Use regex to check if the category starts with the registered key
		// We escape the key to handle special characters and add ^ to match from start
		pattern := "^" + regexp.QuoteMeta(k)
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Errorf("❌ Failed to compile regex for category %q: %v", k, err)
			continue
		}

		if re.MatchString(sanitizedCategory) {
			return v, true
		}
	}

	return nil, false
}

// Combine merges another collection of commands into the receiver.
func (c *Commands) Combine(cmds Commands) *Commands {
	if cmds == nil {
		return c
	}

	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}

		c.Add(cmd)
	}
	return c
}

// Default implements the cmd.Command interface for Netbird commands.
type Default struct {
	cmd      *cobra.Command
	category string
}

// NewDefault creates a new default command runner.
//
// Category Example:
//   - cmd.NewDefault(cmd, "Vault", "Secrets", "Delete") -> "Vault/Secrets/Delete"
func NewDefault(cmd *cobra.Command, categories ...string) Command {
	if len(categories) == 0 {
		_ = fmt.Errorf("❌ error: at least one category is required")
		os.Exit(1)
	}

	// Join all categories with a / to form the category
	category := strings.Join(categories, "/")

	return &Default{
		cmd:      cmd,
		category: category,
	}
}

// Category returns the command category.
func (d *Default) Category() string {
	return d.category
}

// Execute runs the command.
func (d *Default) Execute(choice []string, args []string) bool {
	if len(choice) == 0 {
		log.Warning("Command not implemented yet.")
		return false
	}

	if d.cmd == nil {
		log.Warning("Command not implemented yet.")
		return false
	}
	d.cmd.SetArgs(choice)
	_ = d.cmd.Execute()
	return true
}
