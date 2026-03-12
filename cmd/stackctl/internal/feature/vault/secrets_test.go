package vault

import (
	"testing"
)

func TestResolveSecretValue(t *testing.T) {
	t.Run("given fixed value then returns it", func(t *testing.T) {
		entry := SecretKVEntry{Name: "DB_HOST", Value: "localhost"}

		got, err := ResolveSecretValue(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "localhost" {
			t.Errorf("expected 'localhost', got %q", got)
		}
	})

	t.Run("given auto generate then returns hex with correct length", func(t *testing.T) {
		entry := SecretKVEntry{Name: "API_KEY", AutoGenerate: true, Size: 16}

		got, err := ResolveSecretValue(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 16 bytes = 32 hex chars
		if len(got) != 32 {
			t.Errorf("expected 32 hex chars, got %d: %q", len(got), got)
		}
	})

	t.Run("given auto generate without size then uses default", func(t *testing.T) {
		entry := SecretKVEntry{Name: "TOKEN", AutoGenerate: true}

		got, err := ResolveSecretValue(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// DefaultAutoGenSize=20 bytes = 40 hex chars
		if len(got) != 40 {
			t.Errorf("expected 40 hex chars (default), got %d: %q", len(got), got)
		}
	})

	t.Run("given auto generate then produces unique values", func(t *testing.T) {
		entry := SecretKVEntry{Name: "KEY", AutoGenerate: true, Size: 10}

		val1, _ := ResolveSecretValue(entry)
		val2, _ := ResolveSecretValue(entry)

		if val1 == val2 {
			t.Error("expected unique values, got identical results")
		}
	})
}
