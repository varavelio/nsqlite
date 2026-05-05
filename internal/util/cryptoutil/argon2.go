package cryptoutil

import "github.com/matthewhartstonge/argon2"

// Argon2GenerateHash generates an argon2 hash of the given password.
func Argon2GenerateHash(password string) (string, error) {
	argon := argon2.DefaultConfig()
	hash, err := argon.HashEncoded([]byte(password))
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Argon2CheckHash checks if the given password matches the given argon2 hash.
func Argon2CheckHash(password, hash string) bool {
	ok, err := argon2.VerifyEncoded([]byte(password), []byte(hash))
	if err != nil {
		return false
	}
	return ok
}
