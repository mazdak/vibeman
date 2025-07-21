package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIConfigLoading(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test cases
	tests := []struct {
		name        string
		configToml  string
		wantEnabled bool
		wantImage   string
	}{
		{
			name: "default enabled when no AI section",
			configToml: `
[repository]
name = "test-repo"
`,
			wantEnabled: true,
			wantImage:   "",
		},
		{
			name: "explicit enabled true",
			configToml: `
[repository]
name = "test-repo"

[repository.ai]
enabled = true
`,
			wantEnabled: true,
			wantImage:   "",
		},
		{
			name: "explicit enabled false",
			configToml: `
[repository]
name = "test-repo"

[repository.ai]
enabled = false
`,
			wantEnabled: false,
			wantImage:   "",
		},
		{
			name: "custom image",
			configToml: `
[repository]
name = "test-repo"

[repository.ai]
enabled = true
image = "custom/ai:v2"
`,
			wantEnabled: true,
			wantImage:   "custom/ai:v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			configPath := filepath.Join(tempDir, "vibeman.toml")
			err := os.WriteFile(configPath, []byte(tt.configToml), 0644)
			require.NoError(t, err)

			// Parse config
			config, err := ParseRepositoryConfig(tempDir)
			require.NoError(t, err)

			// Check AI config
			assert.Equal(t, tt.wantEnabled, config.Repository.AI.Enabled)
			assert.Equal(t, tt.wantImage, config.Repository.AI.Image)
		})
	}
}