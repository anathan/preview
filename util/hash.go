package util

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
)

// ComputeHmac256 returns a base64 encoded hash of a message using a secret.
func ComputeHmac256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Hash returns a hex encoded sha1 hash.
func Hash(bytes []byte) string {
	hasher := sha1.New()
	hasher.Write(bytes)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
