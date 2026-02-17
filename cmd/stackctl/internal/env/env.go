package env

import (
	"os"
)

func Get(key string) (string, bool) {
	v := os.Getenv(key)
	return v, v != "" && len(v) > 0
}
