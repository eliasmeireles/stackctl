package list

import (
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

func printSchemas(dbType, dbContext string, schemas []string) {
	items := make([]output.ListItem, len(schemas))
	for i, s := range schemas {
		items[i] = output.NewItem("name", s)
	}
	title := fmt.Sprintf("\n📐 Schemas in %s (%s):", dbType, dbContext)
	if output.IsStructured() {
		title = ""
	}
	output.PrintList(title, []string{"NAME"}, items)
}
