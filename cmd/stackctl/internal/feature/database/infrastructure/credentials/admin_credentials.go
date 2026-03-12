package credentials

type AdminCredentials struct {
	Username  string
	Password  string
	VaultPath string
}

func NewAdminCredentials(username, password string) *AdminCredentials {
	return &AdminCredentials{
		Username: username,
		Password: password,
	}
}

func NewAdminCredentialsWithVault(vaultPath string) *AdminCredentials {
	return &AdminCredentials{
		VaultPath: vaultPath,
	}
}
