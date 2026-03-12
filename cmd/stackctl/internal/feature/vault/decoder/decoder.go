package decoder

// Decoder defines the strategy interface for decoding secret values.
// Implementations can provide different decoding strategies (base64, hex, etc.)
type Decoder interface {
	// Decode transforms the input value according to the strategy.
	// Returns the decoded value or an error if decoding fails.
	Decode(value string) (string, error)
}
