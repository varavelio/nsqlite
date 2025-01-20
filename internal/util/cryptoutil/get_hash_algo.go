package cryptoutil

import (
	"strings"

	"github.com/orsinium-labs/enum"
)

type HashAlgo enum.Member[string]

var (
	HashAlgoPlaintext = HashAlgo{Value: "plaintext"}
	HashAlgoArgon2    = HashAlgo{Value: "argon2"}
	HashAlgoBcrypt    = HashAlgo{Value: "bcrypt"}
)

// GetHashAlgo returns the hash algorithm used for the given token.
func GetHashAlgo(token string) HashAlgo {
	if strings.HasPrefix(token, "$argon2") {
		return HashAlgoArgon2
	}
	if strings.HasPrefix(token, "$2a$") {
		return HashAlgoBcrypt
	}
	return HashAlgoPlaintext
}
