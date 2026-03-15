package list

import (
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

func printUsers(dbType, host string, port int, users []entity.UserInfo) {
	fmt.Printf("\n👥 Users on %s (%s:%d):\n", dbType, host, port)
	if len(users) == 0 {
		fmt.Println("  (no users found)")
		return
	}
	for i, u := range users {
		fmt.Printf("  %d. %s  [%s]\n", i+1, u.Name, u.PermissionsString())
	}
	fmt.Printf("\nTotal: %d user(s)\n", len(users))
}
