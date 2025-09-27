package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		errMsg string
	}{
		{
			name:   "development environment",
			env:    "dev",
			errMsg: "",
		},
		{
			name:   "production environment",
			env:    "prod",
			errMsg: "",
		},
		{
			name:   "empty environment",
			env:    "",
			errMsg: "unknown environment",
		},
		{
			name:   "invalid environment",
			env:    "foo",
			errMsg: "unknown environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			logger, err := New(cfg)

			if tt.errMsg != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errMsg)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}
