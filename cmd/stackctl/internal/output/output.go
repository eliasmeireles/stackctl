// Package output provides structured output formatting for the stackctl CLI.
// Commands use this package to print results in table, JSON, or YAML format
// depending on the --output / -o flag set by the user.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Format represents the output format selected by the user.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Printer writes structured data to an output stream in the configured format.
type Printer struct {
	format Format
	w      io.Writer
}

// global holds the process-wide default printer (set once from root PersistentPreRun).
var global = &Printer{format: FormatTable, w: os.Stdout}

// Set configures the global printer format. Called once from root.go.
func Set(format Format) {
	global = &Printer{format: format, w: os.Stdout}
}

// Get returns the global printer.
func Get() *Printer {
	return global
}

// SetWriter configures the global printer with the given format and writer.
// Intended for use in tests to capture output without touching os.Stdout.
func SetWriter(format Format, w io.Writer) {
	global = &Printer{format: format, w: w}
}

// IsStructured returns true when the current format is not table (i.e. machine-readable).
// Commands can use this to suppress emoji/decorative output.
func IsStructured() bool {
	return global.format != FormatTable
}

// ParseFormat converts a string to a Format, returning an error for unknown values.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return FormatTable, fmt.Errorf("unknown output format %q (valid: table, json, yaml)", s)
	}
}

// --- List ---------------------------------------------------------------

// ListItem represents a single row in a tabular list.
// Keys are preserved in insertion order via the ordered Pairs slice.
type ListItem struct {
	Pairs []Pair
}

// Pair is a single key/value field.
type Pair struct {
	Key   string
	Value string
}

// NewItem builds a ListItem from alternating key, value arguments.
func NewItem(keyValues ...string) ListItem {
	item := ListItem{}
	for i := 0; i+1 < len(keyValues); i += 2 {
		item.Pairs = append(item.Pairs, Pair{Key: keyValues[i], Value: keyValues[i+1]})
	}
	return item
}

// PrintList prints a titled list of items using the global format.
func PrintList(title string, headers []string, items []ListItem) {
	global.PrintList(title, headers, items)
}

func (p *Printer) PrintList(title string, headers []string, items []ListItem) {
	switch p.format {
	case FormatJSON:
		p.printListJSON(items)
	case FormatYAML:
		p.printListYAML(items)
	default:
		p.printListTable(title, headers, items)
	}
}

func (p *Printer) printListTable(title string, headers []string, items []ListItem) {
	if title != "" {
		_, _ = fmt.Fprintln(p.w, title)
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(p.w, "  (none)")
		return
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)

	// header
	if len(headers) > 0 {
		_, _ = fmt.Fprintln(tw, strings.Join(headers, "\t"))
		dividers := make([]string, len(headers))
		for i, h := range headers {
			dividers[i] = strings.Repeat("-", len(h))
		}
		_, _ = fmt.Fprintln(tw, strings.Join(dividers, "\t"))
	}

	for _, item := range items {
		vals := make([]string, len(item.Pairs))
		for i, kv := range item.Pairs {
			vals[i] = kv.Value
		}
		_, _ = fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintf(p.w, "\nTotal: %d\n", len(items))
}

func (p *Printer) printListJSON(items []ListItem) {
	list := make([]map[string]string, len(items))
	for i, item := range items {
		m := make(map[string]string, len(item.Pairs))
		for _, kv := range item.Pairs {
			m[kv.Key] = kv.Value
		}
		list[i] = m
	}
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(list)
}

func (p *Printer) printListYAML(items []ListItem) {
	list := make([]map[string]string, len(items))
	for i, item := range items {
		m := make(map[string]string, len(item.Pairs))
		for _, kv := range item.Pairs {
			m[kv.Key] = kv.Value
		}
		list[i] = m
	}
	b, _ := yaml.Marshal(list)
	_, _ = p.w.Write(b)
}

// --- Record -------------------------------------------------------------

// PrintRecord prints a single record (key/value pairs) using the global format.
func PrintRecord(title string, item ListItem) {
	global.PrintRecord(title, item)
}

func (p *Printer) PrintRecord(title string, item ListItem) {
	switch p.format {
	case FormatJSON:
		m := make(map[string]string, len(item.Pairs))
		for _, kv := range item.Pairs {
			m[kv.Key] = kv.Value
		}
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(m)
	case FormatYAML:
		m := make(map[string]string, len(item.Pairs))
		for _, kv := range item.Pairs {
			m[kv.Key] = kv.Value
		}
		b, _ := yaml.Marshal(m)
		_, _ = p.w.Write(b)
	default:
		if title != "" {
			_, _ = fmt.Fprintln(p.w, title)
		}
		tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)
		for _, kv := range item.Pairs {
			_, _ = fmt.Fprintf(tw, "  %s\t%s\n", kv.Key, kv.Value)
		}
		_ = tw.Flush()
	}
}

// --- Status -------------------------------------------------------------

// StatusResult holds the result of an operation (e.g. test-user).
type StatusResult struct {
	Success bool
	Message string
	Fields  ListItem
}

// PrintStatus prints a success/failure status with optional detail fields.
func PrintStatus(r StatusResult) {
	global.PrintStatus(r)
}

func (p *Printer) PrintStatus(r StatusResult) {
	switch p.format {
	case FormatJSON:
		m := map[string]any{
			"success": r.Success,
			"message": r.Message,
		}
		for _, kv := range r.Fields.Pairs {
			m[kv.Key] = kv.Value
		}
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(m)
	case FormatYAML:
		m := map[string]any{
			"success": r.Success,
			"message": r.Message,
		}
		for _, kv := range r.Fields.Pairs {
			m[kv.Key] = kv.Value
		}
		b, _ := yaml.Marshal(m)
		_, _ = p.w.Write(b)
	default:
		icon := "✅"
		if !r.Success {
			icon = "❌"
		}
		_, _ = fmt.Fprintf(p.w, "%s %s\n", icon, r.Message)
		tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)
		for _, kv := range r.Fields.Pairs {
			_, _ = fmt.Fprintf(tw, "  %s:\t%s\n", kv.Key, kv.Value)
		}
		_ = tw.Flush()
	}
}

// --- Value --------------------------------------------------------------

// PrintValue prints a single named value (e.g. generated password).
func PrintValue(key, value string) {
	global.PrintValue(key, value)
}

func (p *Printer) PrintValue(key, value string) {
	switch p.format {
	case FormatJSON:
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]string{key: value})
	case FormatYAML:
		b, _ := yaml.Marshal(map[string]string{key: value})
		_, _ = p.w.Write(b)
	default:
		_, _ = fmt.Fprintln(p.w, value)
	}
}

// --- Secret map ---------------------------------------------------------

// PrintSecretMap prints a map of secret key/values (e.g. vault secret get).
func PrintSecretMap(path string, data map[string]any) {
	global.PrintSecretMap(path, data)
}

func (p *Printer) PrintSecretMap(path string, data map[string]any) {
	switch p.format {
	case FormatJSON:
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
	case FormatYAML:
		b, _ := yaml.Marshal(data)
		_, _ = p.w.Write(b)
	default:
		b, _ := json.MarshalIndent(data, "", "  ")
		_, _ = fmt.Fprintln(p.w, string(b))
	}
}
