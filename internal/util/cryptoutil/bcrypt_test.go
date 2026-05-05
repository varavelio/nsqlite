package cryptoutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHardcoded(t *testing.T) {
	password := "SecureP@ssw0rd!"
	hash := "$2y$10$y57WjViSJTIayCR.DI8I3OfQ3Tc7fDxIkiGFX/PniqTDuVQm5wdL."

	t.Run("Check Hash", func(t *testing.T) {
		assert.True(t, BcryptCheckHash(password, hash))
	})

	t.Run("Generate And Check Hash", func(t *testing.T) {
		newHash, err := BcryptGenerateHash(password)
		assert.NoError(t, err)
		assert.True(t, BcryptCheckHash(password, newHash))
	})
}

func TestBcryptGenerateHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		cost     []int
		wantErr  bool
	}{
		{"EmptyPassword", "", nil, false},
		{"SimplePassword", "password123", nil, false},
		{"SpecialChars", "P@$$w0rd!", nil, false},
		{"LongPassword", "aVeryLongPasswordThatExceedsNormalLength1234567890", nil, false},
		{"CustomCost", "password", []int{bcrypt.MinCost}, false},
		{
			"InvalidCostTooLow",
			"password",
			[]int{bcrypt.MinCost - 1},
			false,
		}, // Should use default cost
		{
			"InvalidCostTooHigh",
			"password",
			[]int{bcrypt.MaxCost + 1},
			false,
		}, // Should use default cost
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := BcryptGenerateHash(tt.password, tt.cost...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}

func TestBcryptCheckHash(t *testing.T) {
	password := "SecureP@ssw0rd!"
	hash, err := BcryptGenerateHash(password)
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
		{"InvalidHashPrefix", password, "$2x$10$invalidhashvalue", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BcryptCheckHash(tt.password, tt.hash)
			assert.Equal(t, tt.want, result)
		})
	}
}
