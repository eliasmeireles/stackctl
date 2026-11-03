package oidclogin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserCommand(t *testing.T) {
	const target = "https://idp.example.com/auth?state=s1"
	cases := []struct {
		goos, name string
		args       []string
	}{
		{"linux", "xdg-open", []string{target}},
		{"freebsd", "xdg-open", []string{target}},
		{"darwin", "open", []string{target}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", target}},
		{"plan9", "", nil},
	}
	for _, c := range cases {
		t.Run("given "+c.goos+" then it picks the native launcher", func(t *testing.T) {
			name, args := browserCommand(c.goos, target)

			assert.Equal(t, c.name, name)
			assert.Equal(t, c.args, args)
		})
	}
}
