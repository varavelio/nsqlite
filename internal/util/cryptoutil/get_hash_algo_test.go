package cryptoutil

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetHashAlgo(t *testing.T) {
	randomBcrypt, _ := BcryptGenerateHash(uuid.NewString())
	randomArgon2, _ := Argon2GenerateHash(uuid.NewString())

	tests := []struct {
		token string
		want  HashAlgo
	}{
		{
			token: "foobar",
			want:  HashAlgoPlaintext,
		},
		{
			token: randomArgon2,
			want:  HashAlgoArgon2,
		},
		{
			token: "$argon2",
			want:  HashAlgoArgon2,
		},
		{
			token: "$argon2i$v=19$m=16,t=2,p=1$ZGdzZmRzZmdkZnNn$8N0gUF+JbdapSW9dMnHwUg",
			want:  HashAlgoArgon2,
		},
		{
			token: "$argon2d$v=19$m=16,t=2,p=1$ZGdzZmRzZmdkZnNn$34fK63/dgJQ95sMrRop86g",
			want:  HashAlgoArgon2,
		},
		{
			token: "$argon2id$v=19$m=16,t=2,p=1$ZGdzZmRzZmdkZnNn$qx6I7IrSlhqC1Geu3HVrHA",
			want:  HashAlgoArgon2,
		},
		{
			token: randomBcrypt,
			want:  HashAlgoBcrypt,
		},
		{
			token: "$2a$",
			want:  HashAlgoBcrypt,
		},
		{
			token: "$2a$12$G0BlXHbFk2cIlwC9gISF3uS53kzby/WCUSa8XSZq6P.Jc9ADRMD7S",
			want:  HashAlgoBcrypt,
		},
		{
			token: "$2a$15$LA9KV6Jh6a7GqbTT0Z1NfeBVWt3uv2MfvhVa1AdlovB7iw82.O2fO",
			want:  HashAlgoBcrypt,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, GetHashAlgo(test.token))
	}
}
