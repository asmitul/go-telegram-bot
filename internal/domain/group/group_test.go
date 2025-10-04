package group

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup_IsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name        string
		group       *Group
		featureName string
		expected    bool
	}{
		{
			name: "feature enabled explicitly",
			group: &Group{
				Settings: map[string]interface{}{
					"calculator": true,
				},
			},
			featureName: "calculator",
			expected:    true,
		},
		{
			name: "feature disabled explicitly",
			group: &Group{
				Settings: map[string]interface{}{
					"calculator": false,
				},
			},
			featureName: "calculator",
			expected:    false,
		},
		{
			name: "feature not configured - defaults to enabled",
			group: &Group{
				Settings: map[string]interface{}{},
			},
			featureName: "calculator",
			expected:    true,
		},
		{
			name: "feature value is not bool - defaults to enabled",
			group: &Group{
				Settings: map[string]interface{}{
					"calculator": "invalid",
				},
			},
			featureName: "calculator",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.group.IsFeatureEnabled(tt.featureName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGroup_EnableFeature(t *testing.T) {
	g := NewGroup(123, "Test Group", "group")

	// Enable feature
	g.EnableFeature("calculator")

	// Verify
	assert.True(t, g.IsFeatureEnabled("calculator"))
	assert.Equal(t, true, g.Settings["calculator"])
}

func TestGroup_DisableFeature(t *testing.T) {
	g := NewGroup(123, "Test Group", "group")

	// Disable feature
	g.DisableFeature("calculator")

	// Verify
	assert.False(t, g.IsFeatureEnabled("calculator"))
	assert.Equal(t, false, g.Settings["calculator"])
}

func TestGroup_ToggleFeature(t *testing.T) {
	g := NewGroup(123, "Test Group", "group")

	// Initially enabled by default
	assert.True(t, g.IsFeatureEnabled("calculator"))

	// Disable
	g.DisableFeature("calculator")
	assert.False(t, g.IsFeatureEnabled("calculator"))

	// Enable again
	g.EnableFeature("calculator")
	assert.True(t, g.IsFeatureEnabled("calculator"))
}
