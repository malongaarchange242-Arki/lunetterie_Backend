package services

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashFingerprint(template []byte) string {
	hash := sha256.Sum256(template)
	return hex.EncodeToString(hash[:])
}
