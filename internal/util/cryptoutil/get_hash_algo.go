package cryptoutil

import (
	"strings"

	"github.com/orsinium-labs/enum"
)

type HashAlgo enum.Member[string]

var (
	HashAlgoPlaintext = HashAlgo{Value: "plaintext"}
	HashAlgoArgon2ID  = HashAlgo{Value: "argon2id"}
	HashAlgoBcrypt    = HashAlgo{Value: "bcrypt"}
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
