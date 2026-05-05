package cryptoutil

import "golang.org/x/crypto/bcrypt"

// BcryptGenerateHash generates a bcrypt hash of the given password.
//
// The cost parameter is optional. If not provided or if it's not in the valid
// range, the default cost will be used.
func BcryptGenerateHash(password string, cost ...int) (string, error) {
	pickedCost := bcrypt.DefaultCost
	if len(cost) > 0 {
		if cost[0] > bcrypt.MinCost && cost[0] < bcrypt.MaxCost {
			pickedCost = cost[0]
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), pickedCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// BcryptCheckHash checks if the given password matches the given bcrypt hash.
func BcryptCheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
