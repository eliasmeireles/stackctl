package vault

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func (a *Applier) applyPolicies(p *PoliciesConfig) error {
	if len(p.Add) > 0 {
		log.Infof("Adding %d policies", len(p.Add))
		existingPolicies, _ := a.policies.ListPolicies()
		existingMap := make(map[string]bool)
		for _, name := range existingPolicies {
			existingMap[name] = true
		}

		for _, entry := range p.Add {
			if existingMap[entry.Name] {
				log.Infof("⚠️  Policy [%q] already exists. Skipping...", entry.Name)
				continue
			}
			if err := a.writePolicy(entry); err != nil {
				return fmt.Errorf("add policy %q: %w", entry.Name, err)
			}
		}
	}

	if len(p.Update) > 0 {
		log.Infof("Updating %d policies", len(p.Update))
		for _, entry := range p.Update {
			if err := a.writePolicy(entry); err != nil {
				return fmt.Errorf("update policy %q: %w", entry.Name, err)
			}
		}
	}

	if len(p.Delete) > 0 {
		log.Infof("Deleting %d policies", len(p.Delete))
		for _, name := range p.Delete {
			if err := a.policies.DeletePolicy(name); err != nil {
				return fmt.Errorf("delete policy %q: %w", name, err)
			}
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
