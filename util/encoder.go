package util

import "fmt"

func EncodeUrl(b []byte) string {
	allowed := func(c byte) bool {
		if c >= '0' && c <= '9' {
			return true
		}
		if c >= 'A' && c <= 'Z' {
			return true
		}
		if c >= 'a' && c <= 'z' {
			return true
		}
		switch c {
		case '.', '-', '_', '~':
			return true
		}
		return false
	}

	out := make([]byte, 0, len(b)*3)

	for _, c := range b {
		if allowed(c) {
			out = append(out, c)
		} else {
			out = append(out, fmt.Sprintf("%%%02X", c)...)
		}
	}

	return string(out)
}
