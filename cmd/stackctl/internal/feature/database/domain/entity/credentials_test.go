package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentials_Validate(t *testing.T) {
	tests := []struct {
		name    string
		creds   *Credentials
		wantErr bool
	}{
		{
			name: "given valid credentials then validation succeeds",
			creds: &Credentials{
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
			},
			wantErr: false,
		},
		{
			name: "given empty username then validation fails",
			creds: &Credentials{
				Username: "",
				Password: "testpass",
			},
			wantErr: true,
		},
		{
			name: "given empty password then validation fails",
			creds: &Credentials{
				Username: "testuser",
				Password: "",
			},
			wantErr: true,
		},
		{
			name: "given username with leading whitespace then validation fails",
			creds: &Credentials{
				Username: " testuser",
				Password: "testpass",
			},
			wantErr: true,
		},
		{
			name: "given username with trailing whitespace then validation fails",
			creds: &Credentials{
				Username: "testuser ",
				Password: "testpass",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCredentials_HasPrivileges(t *testing.T) {
	tests := []struct {
		name           string
		privileges     []string
		wantPrivileges bool
	}{
		{
			name:           "given privileges then returns true",
			privileges:     []string{"SELECT", "INSERT"},
			wantPrivileges: true,
		},
		{
			name:           "given no privileges then returns false",
			privileges:     []string{},
			wantPrivileges: false,
		},
		{
			name:           "given nil privileges then returns false",
			privileges:     nil,
			wantPrivileges: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &Credentials{
				Username:   "testuser",
				Password:   "testpass",
				Privileges: tt.privileges,
			}
			require.Equal(t, tt.wantPrivileges, creds.HasPrivileges())
		})
	}
}

func TestCredentials_PrivilegesString(t *testing.T) {
	tests := []struct {
		name       string
		privileges []string
		want       string
	}{
		{
			name:       "given multiple privileges then returns comma-separated string",
			privileges: []string{"SELECT", "INSERT", "UPDATE"},
			want:       "SELECT, INSERT, UPDATE",
		},
		{
			name:       "given single privilege then returns single string",
			privileges: []string{"SELECT"},
			want:       "SELECT",
		},
		{
			name:       "given no privileges then returns empty string",
			privileges: []string{},
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &Credentials{
				Username:   "testuser",
				Password:   "testpass",
				Privileges: tt.privileges,
			}
			require.Equal(t, tt.want, creds.PrivilegesString())
		})
	}
}

func TestNewCredentials(t *testing.T) {
	username := "testuser"
	password := "testpass"
	database := "testdb"
	privileges := []string{"SELECT", "INSERT"}

	creds := NewCredentials(username, password, database, privileges)

	require.NotNil(t, creds)
	require.Equal(t, username, creds.Username)
	require.Equal(t, password, creds.Password)
	require.Equal(t, database, creds.Database)
	require.Equal(t, privileges, creds.Privileges)
}
