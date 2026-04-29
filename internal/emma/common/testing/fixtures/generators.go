package fixtures

import (
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
)

// VolumeConfigGen generates random volume configurations for property-based testing
func VolumeConfigGen() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),                    // name
		gen.Identifier(),                    // data_center_id
		gen.IntRange(1, 10000),              // volume_gb
		gen.OneConstOf("ssd", "ssd-plus", "hdd"), // volume_type
	).Map(func(values []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"name":           values[0].(string),
			"data_center_id": values[1].(string),
			"volume_gb":      values[2].(int),
			"volume_type":    values[3].(string),
		}
	})
}

// InvalidVolumeConfigGen generates invalid volume configurations for testing validation
func InvalidVolumeConfigGen() gopter.Gen {
	return gen.OneGenOf(
		// Missing required fields
		gen.Const(map[string]interface{}{
			"name": "test",
		}),
		// Invalid volume size (negative)
		gen.Const(map[string]interface{}{
			"name":           "test",
			"data_center_id": "dc-1",
			"volume_gb":      -1,
			"volume_type":    "ssd",
		}),
		// Invalid volume size (zero)
		gen.Const(map[string]interface{}{
			"name":           "test",
			"data_center_id": "dc-1",
			"volume_gb":      0,
			"volume_type":    "ssd",
		}),
		// Empty volume type
		gen.Const(map[string]interface{}{
			"name":           "test",
			"data_center_id": "dc-1",
			"volume_gb":      100,
			"volume_type":    "",
		}),
		// Empty data center ID
		gen.Const(map[string]interface{}{
			"name":           "test",
			"data_center_id": "",
			"volume_gb":      100,
			"volume_type":    "ssd",
		}),
		// Invalid volume type
		gen.Const(map[string]interface{}{
			"name":           "test",
			"data_center_id": "dc-1",
			"volume_gb":      100,
			"volume_type":    "invalid-type",
		}),
	)
}

// VmConfigGen generates random VM configurations for property-based testing
func VmConfigGen() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),                                  // name
		gen.Identifier(),                                  // data_center_id
		gen.IntRange(1, 100),                              // os_id
		gen.OneConstOf("default", "isolated", "multi-cloud"), // cloud_network_type
		gen.OneConstOf("shared", "standard"),              // vcpu_type
		gen.IntRange(1, 64),                               // vcpu
		gen.IntRange(1, 256),                              // ram_gb
		gen.OneConstOf("ssd", "ssd-plus", "hdd"),          // volume_type
		gen.IntRange(10, 10000),                           // volume_gb
	).Map(func(values []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"name":               values[0].(string),
			"data_center_id":     values[1].(string),
			"os_id":              values[2].(int),
			"cloud_network_type": values[3].(string),
			"vcpu_type":          values[4].(string),
			"vcpu":               values[5].(int),
			"ram_gb":             values[6].(int),
			"volume_type":        values[7].(string),
			"volume_gb":          values[8].(int),
		}
	})
}

// SshKeyConfigGen generates random SSH key configurations for property-based testing
func SshKeyConfigGen() gopter.Gen {
	return gen.OneGenOf(
		// Generated key configuration
		gopter.CombineGens(
			gen.Identifier(),                  // name
			gen.OneConstOf("RSA", "ED25519"),  // key_type
		).Map(func(values []interface{}) map[string]interface{} {
			return map[string]interface{}{
				"name":     values[0].(string),
				"key_type": values[1].(string),
			}
		}),
		// Imported key configuration
		gopter.CombineGens(
			gen.Identifier(),  // name
			gen.AlphaString(), // key (simplified for testing)
		).Map(func(values []interface{}) map[string]interface{} {
			return map[string]interface{}{
				"name": values[0].(string),
				"key":  "ssh-rsa " + values[1].(string) + " test@example.com",
			}
		}),
	)
}

// SecurityGroupConfigGen generates random security group configurations for property-based testing
func SecurityGroupConfigGen() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),          // name
		SecurityGroupRuleGen(),    // rule 1
		SecurityGroupRuleGen(),    // rule 2
	).Map(func(values []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"name":  values[0].(string),
			"rules": []interface{}{values[1], values[2]},
		}
	})
}

// SecurityGroupRuleGen generates random security group rules
func SecurityGroupRuleGen() gopter.Gen {
	return gopter.CombineGens(
		gen.OneConstOf("inbound", "outbound"),                                    // direction
		gen.OneConstOf("tcp", "udp", "icmp", "all", "sctp", "gre", "esp", "ah"), // protocol
		gen.OneGenOf(                                                             // ports
			gen.Const(""),
			gen.Const("80"),
			gen.Const("443"),
			gen.Const("22"),
			gen.Const("80,443"),
			gen.Const("8000-9000"),
		),
		gen.OneGenOf( // ip_range
			gen.Const("0.0.0.0/0"),
			gen.Const("10.0.0.0/8"),
			gen.Const("192.168.1.0/24"),
			gen.Const("172.16.0.0/12"),
		),
	).Map(func(values []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"direction": values[0].(string),
			"protocol":  values[1].(string),
			"ports":     values[2].(string),
			"ip_range":  values[3].(string),
		}
	})
}

// InvalidSshKeyConfigGen generates invalid SSH key configurations for testing validation
func InvalidSshKeyConfigGen() gopter.Gen {
	return gen.OneGenOf(
		// Missing both key and key_type
		gen.Const(map[string]interface{}{
			"name": "test",
		}),
		// Empty name
		gen.Const(map[string]interface{}{
			"name":     "",
			"key_type": "RSA",
		}),
		// Invalid key_type
		gen.Const(map[string]interface{}{
			"name":     "test",
			"key_type": "INVALID",
		}),
		// Both key and key_type specified (mutually exclusive)
		gen.Const(map[string]interface{}{
			"name":     "test",
			"key":      "ssh-rsa AAAA... test@example.com",
			"key_type": "RSA",
		}),
	)
}

// InvalidSecurityGroupConfigGen generates invalid security group configurations for testing validation
func InvalidSecurityGroupConfigGen() gopter.Gen {
	return gen.OneGenOf(
		// Empty name
		gen.Const(map[string]interface{}{
			"name":  "",
			"rules": []interface{}{},
		}),
		// Invalid rule direction
		gen.Const(map[string]interface{}{
			"name": "test",
			"rules": []interface{}{
				map[string]interface{}{
					"direction": "invalid",
					"protocol":  "tcp",
					"ports":     "80",
					"ip_range":  "0.0.0.0/0",
				},
			},
		}),
		// Invalid protocol
		gen.Const(map[string]interface{}{
			"name": "test",
			"rules": []interface{}{
				map[string]interface{}{
					"direction": "inbound",
					"protocol":  "invalid",
					"ports":     "80",
					"ip_range":  "0.0.0.0/0",
				},
			},
		}),
		// Invalid IP range format
		gen.Const(map[string]interface{}{
			"name": "test",
			"rules": []interface{}{
				map[string]interface{}{
					"direction": "inbound",
					"protocol":  "tcp",
					"ports":     "80",
					"ip_range":  "invalid",
				},
			},
		}),
	)
}
