package decoder

// NoOpDecoder implements the Decoder interface with no transformation.
// It returns the input value unchanged (identity function).
type NoOpDecoder struct{}

// NewNoOpDecoder creates a new NoOpDecoder instance.
func NewNoOpDecoder() *NoOpDecoder {
	return &NoOpDecoder{}
}

// Decode returns the input value unchanged.
func (d *NoOpDecoder) Decode(value string) (string, error) {
	return value, nil
}
