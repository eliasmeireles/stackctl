package generator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordGenerator_GeneratePassword(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "given valid size then generates password",
			size:    20,
			wantErr: false,
		},
		{
			name:    "given zero size then uses default size",
			size:    0,
			wantErr: false,
		},
		{
			name:    "given minimum size then generates password",
			size:    8,
			wantErr: false,
		},
		{
			name:    "given maximum size then generates password",
			size:    128,
			wantErr: false,
		},
		{
			name:    "given size below minimum then returns error",
			size:    7,
			wantErr: true,
		},
		{
			name:    "given size above maximum then returns error",
			size:    129,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewPasswordGenerator()
			password, err := generator.GeneratePassword(tt.size)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, password)

			expectedSize := tt.size
			if expectedSize == 0 {
				expectedSize = defaultPasswordSize
			}
			require.Equal(t, expectedSize, len(password))
		})
	}
}

func TestPasswordGenerator_GeneratePassword_Uniqueness(t *testing.T) {
	generator := NewPasswordGenerator()
	generated := make(map[string]bool)

	for i := 0; i < 100; i++ {
		password, err := generator.GeneratePassword(20)
		require.NoError(t, err)
		require.False(t, generated[password], "duplicate password generated")
		generated[password] = true
	}
}

func TestPasswordGenerator_GeneratePassword_Randomness(t *testing.T) {
	generator := NewPasswordGenerator()

	password1, err := generator.GeneratePassword(20)
	require.NoError(t, err)

	password2, err := generator.GeneratePassword(20)
	require.NoError(t, err)

	require.NotEqual(t, password1, password2, "passwords should be different")
}
