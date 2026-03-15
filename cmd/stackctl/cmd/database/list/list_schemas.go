package list

import "fmt"

func printSchemas(dbType, dbContext string, schemas []string) {
	fmt.Printf("\n📐 Schemas in %s (%s):\n", dbType, dbContext)
	if len(schemas) == 0 {
		fmt.Println("  (no schemas found)")
		return
	}
	for i, s := range schemas {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Printf("\nTotal: %d schema(s)\n", len(schemas))
}
