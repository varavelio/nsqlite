package cryptoutil

import "strings"

type HashAlgo string

const (
	HashAlgoPlaintext HashAlgo = "plaintext"
	HashAlgoArgon2ID  HashAlgo = "argon2id"
	HashAlgoBcrypt    HashAlgo = "bcrypt"
)

// GetHashAlgo returns the hash algorithm used for the given token.
func GetHashAlgo(token string) HashAlgo {
	if strings.HasPrefix(token, "$argon2id$") {
		return HashAlgoArgon2ID
	}
	if strings.HasPrefix(token, "$2a$") ||
		strings.HasPrefix(token, "$2b$") ||
		strings.HasPrefix(token, "$2y$") {
		return HashAlgoBcrypt
	}
	return HashAlgoPlaintext
}
