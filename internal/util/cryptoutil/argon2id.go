package cryptoutil

import (
	"github.com/alexedwards/argon2id"
)

// This code uses OWASP recommended parameters for password hashing
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
var argon2idHashParams = &argon2id.Params{
	Memory:      64 * 1024,
	Iterations:  2,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// Argon2IDGenerateHash generates an Argon2id hash of the given password.
//
// This code uses OWASP recommended parameters for password hashing
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
func Argon2IDGenerateHash(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2idHashParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// Argon2IDCheckHash checks if the given password matches the given Argon2id hash.
func Argon2IDCheckHash(password, hash string) bool {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false
	}
	return match
}
