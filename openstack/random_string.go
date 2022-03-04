package openstack

import "crypto/rand"

const (
	NameDefaultSet = `0123456789abcdefghijklmnopqrstuvwxyz`
	PwdDefaultSet  = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz~!@#$%^&*()_+-={}[]:"'<>,./|\'?`
)

func RandomString(charset string, size int) string {
	var bytes = make([]byte, size)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}
