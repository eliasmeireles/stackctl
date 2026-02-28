package vault

import (
	"fmt"
	"os"
)

func (a *Applier) applyPolicies(p *PoliciesConfig) error {
	for _, entry := range p.Add {
		if err := a.writePolicy(entry); err != nil {
			return fmt.Errorf("add policy %q: %w", entry.Name, err)
		}
	}
	for _, entry := range p.Update {
		if err := a.writePolicy(entry); err != nil {
			return fmt.Errorf("update policy %q: %w", entry.Name, err)
		}
	}
	for _, name := range p.Delete {
		if err := a.policies.DeletePolicy(name); err != nil {
			return fmt.Errorf("delete policy %q: %w", name, err)
		}
	}
	return nil
}

func (a *Applier) writePolicy(entry PolicyEntry) error {
	rules, err := ResolvePolicyRules(entry)
	if err != nil {
		return err
	}
	return a.policies.PutPolicy(entry.Name, rules)
}

// ResolvePolicyRules returns the HCL rules for a policy entry,
// reading from file or using inline rules.
func ResolvePolicyRules(entry PolicyEntry) (string, error) {
	if entry.File != "" {
		content, err := os.ReadFile(entry.File)
		if err != nil {
			return "", fmt.Errorf("read policy file %q: %w", entry.File, err)
		}
		return string(content), nil
	}
	if entry.Rules != "" {
		return entry.Rules, nil
	}
	return "", fmt.Errorf("policy %q requires either 'file' or 'rules'", entry.Name)
}
