package cryptoutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgon2IDHardcoded(t *testing.T) {
	password := "SecureP@ssw0rd!"
	hash := "$argon2id$v=19$m=16,t=2,p=1$YmdnaGIzcjQyMzU0d2VyZ2Y$6FgOQLM8ZX1kwlXz4Ekhgw"

	t.Run("Check Hash", func(t *testing.T) {
		assert.True(t, Argon2IDCheckHash(password, hash))
	})

	t.Run("Generate And Check Hash", func(t *testing.T) {
		newHash, err := Argon2IDGenerateHash(password)
		assert.NoError(t, err)
		assert.True(t, Argon2IDCheckHash(password, newHash))
	})
}

func TestArgon2IDGenerateHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"EmptyPassword", "", false},
		{"SimplePassword", "password123", false},
		{"SpecialChars", "P@$$w0rd!", false},
		{"LongPassword", "aVeryLongPasswordThatExceedsNormalLength1234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := Argon2IDGenerateHash(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}

func TestArgon2IDCheckHash(t *testing.T) {
	password := "SecureP@ssw0rd!"
	hash, err := Argon2IDGenerateHash(password)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"CorrectPassword", password, hash, true},
		{"IncorrectPassword", "WrongPassword", hash, false},
		{"EmptyPassword", "", hash, false},
		{"EmptyHash", password, "", false},
		{"InvalidHashFormat", password, "invalidhash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Argon2IDCheckHash(tt.password, tt.hash)
			assert.Equal(t, tt.want, result)
		})
	}
}
