// Package mbtype centralises the small mappings shared across the
// `stackctl messagebroker <broker> ...` command tree.
//
// Mirrors the database/dbtype package — accepts the cmd-string broker form
// (currently only "rabbitmq") and returns the conventional default port.
package mbtype

import "fmt"

// DefaultPort returns the conventional listen port for brokerType.
//
//	rabbitmq → 5672  (AMQP)
func DefaultPort(brokerType string) (int, error) {
	switch brokerType {
	case "rabbitmq":
		return 5672, nil
	default:
		return 0, fmt.Errorf("unsupported message broker type: %s (supported: rabbitmq)", brokerType)
	}
}

// ApplyDefaultPort sets *port to the default for brokerType when *port is zero.
// Returns an error when brokerType is not supported. No-op when *port is already set.
func ApplyDefaultPort(brokerType string, port *int) error {
	if *port != 0 {
		return nil
	}
	p, err := DefaultPort(brokerType)
	if err != nil {
		return err
	}
	*port = p
	return nil
}
