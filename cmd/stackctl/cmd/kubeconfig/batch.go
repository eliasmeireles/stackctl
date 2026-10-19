package kubeconfig

import (
	"fmt"
	"strings"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// batch applies one operation to several names and reports a status per name.
type batch struct {
	title  string
	done   string
	pathOf func(name string) string
	apply  func(name string) error
}

// run applies the operation to every name, even after failures, prints the
// result table and returns an error listing the names that failed.
func (b batch) run(names []string) error {
	items := make([]output.ListItem, 0, len(names))
	var failed []string
	for _, name := range names {
		status := b.done
		if err := b.apply(name); err != nil {
			status = "failed: " + err.Error()
			failed = append(failed, name)
		}
		items = append(items, b.item(name, status))
	}
	output.PrintList(b.title, b.headers(), items)
	if len(failed) > 0 {
		return fmt.Errorf("%d of %d failed: %s", len(failed), len(names), strings.Join(failed, ", "))
	}
	return nil
}

func (b batch) headers() []string {
	if b.pathOf == nil {
		return []string{"NAME", "STATUS"}
	}
	return []string{"NAME", "PATH", "STATUS"}
}

func (b batch) item(name, status string) output.ListItem {
	if b.pathOf == nil {
		return output.NewItem("name", name, "status", status)
	}
	return output.NewItem("name", name, "path", b.pathOf(name), "status", status)
}
