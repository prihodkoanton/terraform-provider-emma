package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVolumeFixture(t *testing.T) {
	volume := VolumeFixture()

	// Test that fixture returns valid data
	assert.NotNil(t, volume, "Volume fixture should not be nil")
	assert.NotNil(t, volume.Id, "Volume ID should not be nil")
	assert.Equal(t, int32(12345), *volume.Id, "Volume ID should match")
	assert.NotNil(t, volume.Name, "Volume name should not be nil")
	assert.Equal(t, "test-volume", *volume.Name, "Volume name should match")
	assert.NotNil(t, volume.SizeGb, "Volume size should not be nil")
	assert.Equal(t, int32(100), *volume.SizeGb, "Volume size should match")
	assert.NotNil(t, volume.Type, "Volume type should not be nil")
	assert.Equal(t, "ssd", *volume.Type, "Volume type should match")
	assert.NotNil(t, volume.IsSystem, "IsSystem should not be nil")
	assert.False(t, *volume.IsSystem, "IsSystem should be false")
	assert.NotNil(t, volume.Status, "Status should not be nil")
	assert.Equal(t, "available", *volume.Status, "Status should match")

	// Test provider info
	provider, ok := volume.GetProviderOk()
	assert.True(t, ok, "Provider should be set")
	assert.NotNil(t, provider.Id, "Provider ID should not be nil")
	assert.Equal(t, int32(1), *provider.Id, "Provider ID should match")
	assert.NotNil(t, provider.Name, "Provider name should not be nil")
	assert.Equal(t, "AWS", *provider.Name, "Provider name should match")

	// Test location info
	location, ok := volume.GetLocationOk()
	assert.True(t, ok, "Location should be set")
	assert.NotNil(t, location.Id, "Location ID should not be nil")
	assert.Equal(t, int32(1), *location.Id, "Location ID should match")

	// Test data center info
	dataCenter, ok := volume.GetDataCenterOk()
	assert.True(t, ok, "Data center should be set")
	assert.NotNil(t, dataCenter.Id, "Data center ID should not be nil")
	assert.Equal(t, "dc-test-1", *dataCenter.Id, "Data center ID should match")
}

func TestVolumeFixtureConsistency(t *testing.T) {
	// Test that multiple calls return consistent data
	volume1 := VolumeFixture()
	volume2 := VolumeFixture()

	assert.Equal(t, volume1.Id, volume2.Id, "Volume IDs should be consistent")
	assert.Equal(t, volume1.Name, volume2.Name, "Volume names should be consistent")
	assert.Equal(t, volume1.SizeGb, volume2.SizeGb, "Volume sizes should be consistent")
	assert.Equal(t, volume1.Type, volume2.Type, "Volume types should be consistent")
}

func TestVmFixture(t *testing.T) {
	vm := VmFixture()

	// Test that fixture returns valid data
	assert.NotNil(t, vm, "VM fixture should not be nil")
	assert.NotNil(t, vm.Id, "VM ID should not be nil")
	assert.Equal(t, int32(67890), *vm.Id, "VM ID should match")
	assert.NotNil(t, vm.Name, "VM name should not be nil")
	assert.Equal(t, "test-vm", *vm.Name, "VM name should match")
	assert.NotNil(t, vm.VCpu, "VM vCPU should not be nil")
	assert.Equal(t, int32(2), *vm.VCpu, "VM vCPU should match")
	assert.NotNil(t, vm.RamGb, "VM RAM should not be nil")
	assert.Equal(t, int32(4), *vm.RamGb, "VM RAM should match")
	assert.NotNil(t, vm.Status, "VM status should not be nil")
	assert.Equal(t, "running", *vm.Status, "VM status should match")

	// Test disks
	disks, ok := vm.GetDisksOk()
	assert.True(t, ok, "Disks should be set")
	assert.Len(t, disks, 1, "Should have one disk")
	assert.NotNil(t, disks[0].Id, "Disk ID should not be nil")
	assert.Equal(t, int32(1), *disks[0].Id, "Disk ID should match")

	// Test networks
	networks, ok := vm.GetNetworksOk()
	assert.True(t, ok, "Networks should be set")
	assert.Len(t, networks, 1, "Should have one network")
	assert.NotNil(t, networks[0].Ip, "Network IP should not be nil")
	assert.Equal(t, "10.0.0.1", *networks[0].Ip, "Network IP should match")

	// Test cost
	cost, ok := vm.GetCostOk()
	assert.True(t, ok, "Cost should be set")
	assert.NotNil(t, cost.Unit, "Cost unit should not be nil")
	assert.Equal(t, "month", *cost.Unit, "Cost unit should match")
	assert.NotNil(t, cost.Price, "Cost price should not be nil")
	assert.Equal(t, float32(50.0), *cost.Price, "Cost price should match")
}

func TestVmFixtureConsistency(t *testing.T) {
	// Test that multiple calls return consistent data
	vm1 := VmFixture()
	vm2 := VmFixture()

	assert.Equal(t, vm1.Id, vm2.Id, "VM IDs should be consistent")
	assert.Equal(t, vm1.Name, vm2.Name, "VM names should be consistent")
	assert.Equal(t, vm1.VCpu, vm2.VCpu, "VM vCPUs should be consistent")
	assert.Equal(t, vm1.RamGb, vm2.RamGb, "VM RAM should be consistent")
}

func TestSshKeyFixture(t *testing.T) {
	sshKey := SshKeyFixture()

	// Test that fixture returns valid data
	assert.NotNil(t, sshKey, "SSH key fixture should not be nil")
	assert.NotNil(t, sshKey.Id, "SSH key ID should not be nil")
	assert.Equal(t, int32(11111), *sshKey.Id, "SSH key ID should match")
	assert.NotNil(t, sshKey.Name, "SSH key name should not be nil")
	assert.Equal(t, "test-ssh-key", *sshKey.Name, "SSH key name should match")
	assert.NotNil(t, sshKey.Key, "SSH key should not be nil")
	assert.Contains(t, *sshKey.Key, "ssh-rsa", "SSH key should contain ssh-rsa")
	assert.NotNil(t, sshKey.Fingerprint, "SSH key fingerprint should not be nil")
	assert.Contains(t, *sshKey.Fingerprint, "SHA256:", "Fingerprint should contain SHA256:")
	assert.NotNil(t, sshKey.KeyType, "SSH key type should not be nil")
	assert.Equal(t, "RSA", *sshKey.KeyType, "SSH key type should match")
}

func TestSshKeyFixtureConsistency(t *testing.T) {
	// Test that multiple calls return consistent data
	sshKey1 := SshKeyFixture()
	sshKey2 := SshKeyFixture()

	assert.Equal(t, sshKey1.Id, sshKey2.Id, "SSH key IDs should be consistent")
	assert.Equal(t, sshKey1.Name, sshKey2.Name, "SSH key names should be consistent")
	assert.Equal(t, sshKey1.Key, sshKey2.Key, "SSH keys should be consistent")
	assert.Equal(t, sshKey1.Fingerprint, sshKey2.Fingerprint, "SSH key fingerprints should be consistent")
}

func TestSecurityGroupFixture(t *testing.T) {
	securityGroup := SecurityGroupFixture()

	// Test that fixture returns valid data
	assert.NotNil(t, securityGroup, "Security group fixture should not be nil")
	assert.NotNil(t, securityGroup.Id, "Security group ID should not be nil")
	assert.Equal(t, int32(22222), *securityGroup.Id, "Security group ID should match")
	assert.NotNil(t, securityGroup.Name, "Security group name should not be nil")
	assert.Equal(t, "test-security-group", *securityGroup.Name, "Security group name should match")
	assert.NotNil(t, securityGroup.SynchronizationStatus, "Synchronization status should not be nil")
	assert.Equal(t, "Synchronized", *securityGroup.SynchronizationStatus, "Synchronization status should match")
	assert.NotNil(t, securityGroup.RecomposingStatus, "Recomposing status should not be nil")
	assert.Equal(t, "Idle", *securityGroup.RecomposingStatus, "Recomposing status should match")

	// Test rules
	rules, ok := securityGroup.GetRulesOk()
	assert.True(t, ok, "Rules should be set")
	assert.Len(t, rules, 2, "Should have two rules")

	// Test first rule (inbound)
	assert.NotNil(t, rules[0].Direction, "Rule direction should not be nil")
	assert.Equal(t, "inbound", *rules[0].Direction, "Rule direction should match")
	assert.NotNil(t, rules[0].Protocol, "Rule protocol should not be nil")
	assert.Equal(t, "tcp", *rules[0].Protocol, "Rule protocol should match")
	assert.NotNil(t, rules[0].Ports, "Rule ports should not be nil")
	assert.Equal(t, "80,443", *rules[0].Ports, "Rule ports should match")

	// Test second rule (outbound)
	assert.NotNil(t, rules[1].Direction, "Rule direction should not be nil")
	assert.Equal(t, "outbound", *rules[1].Direction, "Rule direction should match")
	assert.NotNil(t, rules[1].Protocol, "Rule protocol should not be nil")
	assert.Equal(t, "all", *rules[1].Protocol, "Rule protocol should match")
}

func TestSecurityGroupFixtureConsistency(t *testing.T) {
	// Test that multiple calls return consistent data
	securityGroup1 := SecurityGroupFixture()
	securityGroup2 := SecurityGroupFixture()

	assert.Equal(t, securityGroup1.Id, securityGroup2.Id, "Security group IDs should be consistent")
	assert.Equal(t, securityGroup1.Name, securityGroup2.Name, "Security group names should be consistent")
	assert.Equal(t, securityGroup1.SynchronizationStatus, securityGroup2.SynchronizationStatus, "Synchronization statuses should be consistent")
}
