package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		envVars map[string]string
		want    AppConfig
	}{
		{
			name: "Default values",
			args: []string{},
			want: AppConfig{
				Verbose:    true,            // Default is true
				Timeout:    5 * time.Second, // Default is 5s
				ConfigType: "gitlab",        // Default is gitlab
				Output:     "/tmp/data",
				Split:      30,
				AlertDays:  5,
			},
		},
		{
			name: "Flags override defaults",
			args: []string{"-timeout", "10s", "-verbose=false", "-type", "zone"},
			want: AppConfig{
				Verbose:    false,
				Timeout:    10 * time.Second,
				ConfigType: "zone",
				Output:     "/tmp/data", // Default remains
				Split:      30,
				AlertDays:  5,
			},
		},
		{
			name: "Env Vars work",
			args: []string{},
			envVars: map[string]string{
				"SSL_TIMEOUT":    "20s",
				"SSL_OUTPUTFILE": "/custom/path",
			},
			want: AppConfig{
				Verbose:    true,
				Timeout:    20 * time.Second,
				Output:     "/custom/path",
				ConfigType: "gitlab",
				Split:      30,
				AlertDays:  5,
			},
		},
		{
			name: "Flags take precedence over Env Vars",
			args: []string{"-timeout", "5s"}, // Flag says 5s
			envVars: map[string]string{
				"SSL_TIMEOUT": "99s", // Env says 99s
			},
			want: AppConfig{
				Timeout:    5 * time.Second, // Flag should win
				Verbose:    true,
				ConfigType: "gitlab",
				Output:     "/tmp/data",
				Split:      30,
				AlertDays:  5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Set Environment Variables for this test case
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// 2. Call Load with simulated CLI args
			got, err := Load(tt.args)

			// 3. Assertions
			assert.NoError(t, err)

			// We compare specific fields because zero-values in 'want'
			// might match fields we aren't testing, making DeepEqual tricky.
			// Using testify/assert makes this readable.
			assert.Equal(t, tt.want.Timeout, got.Timeout, "Timeout mismatch")
			assert.Equal(t, tt.want.Verbose, got.Verbose, "Verbose mismatch")
			assert.Equal(t, tt.want.ConfigType, got.ConfigType, "ConfigType mismatch")
			assert.Equal(t, tt.want.Output, got.Output, "Output mismatch")
		})
	}
}
