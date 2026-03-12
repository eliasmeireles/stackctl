package credentials

type AdminCredentials struct {
	Username string
	Password string
}

func NewAdminCredentials(username, password string) *AdminCredentials {
	return &AdminCredentials{
		Username: username,
		Password: password,
	}
}
