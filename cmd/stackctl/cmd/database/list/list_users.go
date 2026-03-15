package list

import (
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

func printUsers(dbType, host string, port int, users []entity.UserInfo) {
	items := make([]output.ListItem, len(users))
	for i, u := range users {
		items[i] = output.NewItem("name", u.Name, "permissions", u.PermissionsString())
	}
	title := fmt.Sprintf("\n👥 Users on %s (%s:%d):", dbType, host, port)
	if output.IsStructured() {
		title = ""
	}
	output.PrintList(title, []string{"NAME", "PERMISSIONS"}, items)
}
