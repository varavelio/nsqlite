package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_validateTransactionTimeout(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantErr  bool
	}{
		{
			name:     "valid - 1 second",
			duration: time.Second,
			wantErr:  false,
		},
		{
			name:     "valid - 1 minute",
			duration: time.Minute,
			wantErr:  false,
		},
		{
			name:     "invalid - negative",
			duration: -time.Second,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransactionTimeout(tt.duration)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
