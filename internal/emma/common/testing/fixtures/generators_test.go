package fixtures

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

func TestVolumeConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates valid volume configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Test that all required fields are present
			_, hasName := config["name"]
			_, hasDataCenterId := config["data_center_id"]
			_, hasVolumeGb := config["volume_gb"]
			_, hasVolumeType := config["volume_type"]

			if !hasName || !hasDataCenterId || !hasVolumeGb || !hasVolumeType {
				return false
			}

			// Test that volume_gb is in valid range
			volumeGb, ok := config["volume_gb"].(int)
			if !ok || volumeGb < 1 || volumeGb > 10000 {
				return false
			}

			// Test that volume_type is valid
			volumeType, ok := config["volume_type"].(string)
			if !ok {
				return false
			}
			validTypes := map[string]bool{"ssd": true, "ssd-plus": true, "hdd": true}
			return validTypes[volumeType]
		},
		VolumeConfigGen(),
	))

	properties.TestingRun(t)
}

func TestInvalidVolumeConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates invalid volume configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Each invalid config should have at least one invalid aspect
			// We just verify that the generator produces configs
			return config != nil
		},
		InvalidVolumeConfigGen(),
	))

	properties.TestingRun(t)
}

func TestVmConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates valid VM configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Test that all required fields are present
			requiredFields := []string{
				"name", "data_center_id", "os_id", "cloud_network_type",
				"vcpu_type", "vcpu", "ram_gb", "volume_type", "volume_gb",
			}

			for _, field := range requiredFields {
				if _, exists := config[field]; !exists {
					return false
				}
			}

			// Test that numeric fields are in valid ranges
			vcpu, ok := config["vcpu"].(int)
			if !ok || vcpu < 1 || vcpu > 64 {
				return false
			}

			ramGb, ok := config["ram_gb"].(int)
			if !ok || ramGb < 1 || ramGb > 256 {
				return false
			}

			volumeGb, ok := config["volume_gb"].(int)
			if !ok || volumeGb < 10 || volumeGb > 10000 {
				return false
			}

			return true
		},
		VmConfigGen(),
	))

	properties.TestingRun(t)
}

func TestSshKeyConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates valid SSH key configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Test that name is present
			_, hasName := config["name"]
			if !hasName {
				return false
			}

			// Test that either key or key_type is present (but not both)
			_, hasKey := config["key"]
			_, hasKeyType := config["key_type"]

			// Valid if exactly one is present
			return (hasKey && !hasKeyType) || (!hasKey && hasKeyType)
		},
		SshKeyConfigGen(),
	))

	properties.TestingRun(t)
}

func TestSecurityGroupConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates valid security group configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Test that name is present
			_, hasName := config["name"]
			if !hasName {
				return false
			}

			// Test that rules are present and valid
			rules, hasRules := config["rules"]
			if !hasRules {
				return false
			}

			rulesSlice, ok := rules.([]interface{})
			if !ok {
				return false
			}

			// Test that we have at least one rule
			if len(rulesSlice) < 1 {
				return false
			}

			// Test that each rule has required fields
			for _, rule := range rulesSlice {
				ruleMap, ok := rule.(map[string]interface{})
				if !ok {
					return false
				}

				requiredFields := []string{"direction", "protocol", "ports", "ip_range"}
				for _, field := range requiredFields {
					if _, exists := ruleMap[field]; !exists {
						return false
					}
				}
			}

			return true
		},
		SecurityGroupConfigGen(),
	))

	properties.TestingRun(t)
}

func TestInvalidSshKeyConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates invalid SSH key configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Each invalid config should have at least one invalid aspect
			// We just verify that the generator produces configs
			return config != nil
		},
		InvalidSshKeyConfigGen(),
	))

	properties.TestingRun(t)
}

func TestInvalidSecurityGroupConfigGen(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("generates invalid security group configurations", prop.ForAll(
		func(config map[string]interface{}) bool {
			// Each invalid config should have at least one invalid aspect
			// We just verify that the generator produces configs
			return config != nil
		},
		InvalidSecurityGroupConfigGen(),
	))

	properties.TestingRun(t)
}

// Unit tests to verify generator variety

func TestVolumeConfigGenVariety(t *testing.T) {
	// Generate multiple configs and verify they're different
	configs := make([]map[string]interface{}, 5)
	params := gopter.DefaultGenParameters()
	for i := 0; i < 5; i++ {
		result, ok := VolumeConfigGen()(params).Retrieve()
		assert.True(t, ok, "Generator should produce a value")
		configs[i] = result.(map[string]interface{})
	}

	// Check that at least some configs are different
	allSame := true
	for i := 1; i < len(configs); i++ {
		if configs[i]["name"] != configs[0]["name"] ||
			configs[i]["volume_gb"] != configs[0]["volume_gb"] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "Generator should produce varied configurations")
}

func TestInvalidVolumeConfigGenProducesInvalidData(t *testing.T) {
	// Generate an invalid config and verify it has at least one invalid aspect
	params := gopter.DefaultGenParameters()
	result, ok := InvalidVolumeConfigGen()(params).Retrieve()
	assert.True(t, ok, "Generator should produce a value")

	config := result.(map[string]interface{})
	assert.NotNil(t, config, "Config should not be nil")

	// Check for various invalid conditions
	hasInvalidAspect := false

	// Missing required fields
	if _, hasName := config["name"]; !hasName {
		hasInvalidAspect = true
	}
	if _, hasDataCenterId := config["data_center_id"]; !hasDataCenterId {
		hasInvalidAspect = true
	}
	if _, hasVolumeGb := config["volume_gb"]; !hasVolumeGb {
		hasInvalidAspect = true
	}
	if _, hasVolumeType := config["volume_type"]; !hasVolumeType {
		hasInvalidAspect = true
	}

	// Invalid values
	if volumeGb, ok := config["volume_gb"].(int); ok && volumeGb <= 0 {
		hasInvalidAspect = true
	}
	if volumeType, ok := config["volume_type"].(string); ok {
		if volumeType == "" || volumeType == "invalid-type" {
			hasInvalidAspect = true
		}
	}
	if dataCenterId, ok := config["data_center_id"].(string); ok && dataCenterId == "" {
		hasInvalidAspect = true
	}
	// Missing required fields
	if _, hasName := config["name"]; !hasName {
		hasInvalidAspect = true
	}

	assert.True(t, hasInvalidAspect, "Invalid config should have at least one invalid aspect")
}

func TestVmConfigGenProducesValidData(t *testing.T) {
	// Generate a VM config and verify it's valid
	params := gopter.DefaultGenParameters()
	result, ok := VmConfigGen()(params).Retrieve()
	assert.True(t, ok, "Generator should produce a value")

	config := result.(map[string]interface{})
	assert.NotNil(t, config, "Config should not be nil")

	// Verify all required fields are present
	assert.Contains(t, config, "name")
	assert.Contains(t, config, "data_center_id")
	assert.Contains(t, config, "os_id")
	assert.Contains(t, config, "cloud_network_type")
	assert.Contains(t, config, "vcpu_type")
	assert.Contains(t, config, "vcpu")
	assert.Contains(t, config, "ram_gb")
	assert.Contains(t, config, "volume_type")
	assert.Contains(t, config, "volume_gb")

	// Verify numeric fields are in valid ranges
	vcpu := config["vcpu"].(int)
	assert.GreaterOrEqual(t, vcpu, 1)
	assert.LessOrEqual(t, vcpu, 64)

	ramGb := config["ram_gb"].(int)
	assert.GreaterOrEqual(t, ramGb, 1)
	assert.LessOrEqual(t, ramGb, 256)
}

func TestSshKeyConfigGenProducesValidData(t *testing.T) {
	// Generate multiple SSH key configs and verify they're valid
	params := gopter.DefaultGenParameters()
	for i := 0; i < 5; i++ {
		result, ok := SshKeyConfigGen()(params).Retrieve()
		assert.True(t, ok, "Generator should produce a value")

		config := result.(map[string]interface{})
		assert.NotNil(t, config, "Config should not be nil")
		assert.Contains(t, config, "name")

		// Verify exactly one of key or key_type is present
		_, hasKey := config["key"]
		_, hasKeyType := config["key_type"]
		assert.True(t, (hasKey && !hasKeyType) || (!hasKey && hasKeyType),
			"Config should have either key or key_type, but not both")
	}
}
