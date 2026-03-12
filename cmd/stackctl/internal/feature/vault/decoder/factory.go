package decoder

// Type represents the type of decoder to create.
type Type string

const (
	// TypeBase64 represents base64 decoding strategy.
	TypeBase64 Type = "base64"
	// TypeNone represents no decoding (identity function).
	TypeNone Type = "none"
)

// Factory creates decoder instances based on the specified type.
type Factory struct{}

// NewFactory creates a new decoder factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Create returns a decoder instance based on the specified type.
func (f *Factory) Create(decoderType Type) Decoder {
	switch decoderType {
	case TypeBase64:
		return NewBase64Decoder()
	default:
		return NewNoOpDecoder()
	}
}

// CreateFromFlag creates a decoder based on a boolean flag.
// If decodeBase64 is true, returns Base64Decoder, otherwise NoOpDecoder.
func (f *Factory) CreateFromFlag(decodeBase64 bool) Decoder {
	if decodeBase64 {
		return f.Create(TypeBase64)
	}
	return f.Create(TypeNone)
}
