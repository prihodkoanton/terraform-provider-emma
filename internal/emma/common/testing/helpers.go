package testing

import (
	"context"
	"testing"

	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestContext creates a context for testing
func TestContext() context.Context {
	return context.Background()
}

// AssertStringEqual checks if two strings are equal
func AssertStringEqual(t *testing.T, expected, actual, fieldName string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected '%s', got '%s'", fieldName, expected, actual)
	}
}

// AssertInt32Equal checks if two int32 values are equal
func AssertInt32Equal(t *testing.T, expected, actual int32, fieldName string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %d, got %d", fieldName, expected, actual)
	}
}

// AssertInt64Equal checks if two int64 values are equal
func AssertInt64Equal(t *testing.T, expected, actual int64, fieldName string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %d, got %d", fieldName, expected, actual)
	}
}

// AssertBoolEqual checks if two bool values are equal
func AssertBoolEqual(t *testing.T, expected, actual bool, fieldName string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", fieldName, expected, actual)
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, value interface{}, fieldName string) {
	t.Helper()
	if value == nil {
		t.Errorf("%s: expected non-nil value, got nil", fieldName)
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, value interface{}, fieldName string) {
	t.Helper()
	if value != nil {
		t.Errorf("%s: expected nil value, got %v", fieldName, value)
	}
}

// AssertTerraformStringEqual checks if a Terraform string value matches expected
func AssertTerraformStringEqual(t *testing.T, expected string, actual types.String, fieldName string) {
	t.Helper()
	if actual.IsNull() {
		t.Errorf("%s: expected '%s', got null", fieldName, expected)
		return
	}
	if actual.IsUnknown() {
		t.Errorf("%s: expected '%s', got unknown", fieldName, expected)
		return
	}
	if actual.ValueString() != expected {
		t.Errorf("%s: expected '%s', got '%s'", fieldName, expected, actual.ValueString())
	}
}

// AssertTerraformInt64Equal checks if a Terraform int64 value matches expected
func AssertTerraformInt64Equal(t *testing.T, expected int64, actual types.Int64, fieldName string) {
	t.Helper()
	if actual.IsNull() {
		t.Errorf("%s: expected %d, got null", fieldName, expected)
		return
	}
	if actual.IsUnknown() {
		t.Errorf("%s: expected %d, got unknown", fieldName, expected)
		return
	}
	if actual.ValueInt64() != expected {
		t.Errorf("%s: expected %d, got %d", fieldName, expected, actual.ValueInt64())
	}
}

// AssertTerraformBoolEqual checks if a Terraform bool value matches expected
func AssertTerraformBoolEqual(t *testing.T, expected bool, actual types.Bool, fieldName string) {
	t.Helper()
	if actual.IsNull() {
		t.Errorf("%s: expected %v, got null", fieldName, expected)
		return
	}
	if actual.IsUnknown() {
		t.Errorf("%s: expected %v, got unknown", fieldName, expected)
		return
	}
	if actual.ValueBool() != expected {
		t.Errorf("%s: expected %v, got %v", fieldName, expected, actual.ValueBool())
	}
}

// AssertTerraformStringNull checks if a Terraform string value is null
func AssertTerraformStringNull(t *testing.T, actual types.String, fieldName string) {
	t.Helper()
	if !actual.IsNull() {
		t.Errorf("%s: expected null, got '%s'", fieldName, actual.ValueString())
	}
}

// AssertTerraformInt64Null checks if a Terraform int64 value is null
func AssertTerraformInt64Null(t *testing.T, actual types.Int64, fieldName string) {
	t.Helper()
	if !actual.IsNull() {
		t.Errorf("%s: expected null, got %d", fieldName, actual.ValueInt64())
	}
}

// AssertTerraformBoolNull checks if a Terraform bool value is null
func AssertTerraformBoolNull(t *testing.T, actual types.Bool, fieldName string) {
	t.Helper()
	if !actual.IsNull() {
		t.Errorf("%s: expected null, got %v", fieldName, actual.ValueBool())
	}
}

// AssertTerraformObjectNotNull checks if a Terraform object is not null
func AssertTerraformObjectNotNull(t *testing.T, actual types.Object, fieldName string) {
	t.Helper()
	if actual.IsNull() {
		t.Errorf("%s: expected non-null object, got null", fieldName)
	}
	if actual.IsUnknown() {
		t.Errorf("%s: expected non-null object, got unknown", fieldName)
	}
}

// AssertTerraformListNotNull checks if a Terraform list is not null
func AssertTerraformListNotNull(t *testing.T, actual types.List, fieldName string) {
	t.Helper()
	if actual.IsNull() {
		t.Errorf("%s: expected non-null list, got null", fieldName)
	}
	if actual.IsUnknown() {
		t.Errorf("%s: expected non-null list, got unknown", fieldName)
	}
}

// AssertNoError checks if an error is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// AssertError checks if an error is not nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// AssertErrorContains checks if an error contains a specific message
func AssertErrorContains(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error containing '%s', got nil", expectedMessage)
		return
	}
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("expected error to contain '%s', got '%s'", expectedMessage, err.Error())
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring finds a substring in a string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MockVolumeBuilder builds mock volume data for testing
type MockVolumeBuilder struct {
	volume *emmaSdk.Volume
}

// NewMockVolumeBuilder creates a new mock volume builder with defaults
func NewMockVolumeBuilder() *MockVolumeBuilder {
	id := int32(12345)
	name := "test-volume"
	sizeGb := int32(100)
	volumeType := "ssd"
	isSystem := false
	status := "available"
	projectId := int32(1)
	createdAt := "2025-02-03T10:00:00Z"

	volume := &emmaSdk.Volume{}
	volume.SetId(id)
	volume.SetName(name)
	volume.SetSizeGb(sizeGb)
	volume.SetType(volumeType)
	volume.SetIsSystem(isSystem)
	volume.SetStatus(status)
	volume.SetProjectId(projectId)
	volume.SetCreatedAt(createdAt)

	// Set default provider
	provider := emmaSdk.NewVolumeProvider()
	provider.SetId(1)
	provider.SetName("AWS")
	volume.SetProvider(*provider)

	// Set default location
	location := emmaSdk.NewVolumeLocation()
	location.SetId(1)
	location.SetName("US East")
	location.SetContinent("North America")
	location.SetRegion("us-east-1")
	volume.SetLocation(*location)

	// Set default data center
	dataCenter := emmaSdk.NewVolumeDataCenter()
	dataCenter.SetId("dc-test-1")
	dataCenter.SetName("Test Data Center")
	volume.SetDataCenter(*dataCenter)

	return &MockVolumeBuilder{volume: volume}
}

// WithId sets the volume ID
func (b *MockVolumeBuilder) WithId(id int32) *MockVolumeBuilder {
	b.volume.SetId(id)
	return b
}

// WithName sets the volume name
func (b *MockVolumeBuilder) WithName(name string) *MockVolumeBuilder {
	b.volume.SetName(name)
	return b
}

// WithSizeGb sets the volume size
func (b *MockVolumeBuilder) WithSizeGb(sizeGb int32) *MockVolumeBuilder {
	b.volume.SetSizeGb(sizeGb)
	return b
}

// WithType sets the volume type
func (b *MockVolumeBuilder) WithType(volumeType string) *MockVolumeBuilder {
	b.volume.SetType(volumeType)
	return b
}

// WithStatus sets the volume status
func (b *MockVolumeBuilder) WithStatus(status string) *MockVolumeBuilder {
	b.volume.SetStatus(status)
	return b
}

// WithAttachedToId sets the attached instance ID
func (b *MockVolumeBuilder) WithAttachedToId(attachedToId int32) *MockVolumeBuilder {
	b.volume.SetAttachedToId(attachedToId)
	return b
}

// WithProjectId sets the project ID
func (b *MockVolumeBuilder) WithProjectId(projectId int32) *MockVolumeBuilder {
	b.volume.SetProjectId(projectId)
	return b
}

// WithDataCenterId sets the data center ID
func (b *MockVolumeBuilder) WithDataCenterId(dataCenterId string) *MockVolumeBuilder {
	dataCenter := emmaSdk.NewVolumeDataCenter()
	dataCenter.SetId(dataCenterId)
	dataCenter.SetName("Data Center " + dataCenterId)
	b.volume.SetDataCenter(*dataCenter)
	return b
}

// Build returns the built volume
func (b *MockVolumeBuilder) Build() *emmaSdk.Volume {
	return b.volume
}

// MockVmBuilder builds mock VM data for testing
type MockVmBuilder struct {
	vm *emmaSdk.Vm
}

// NewMockVmBuilder creates a new mock VM builder with defaults
func NewMockVmBuilder() *MockVmBuilder {
	id := int32(67890)
	name := "test-vm"
	vcpu := int32(2)
	ramGb := int32(4)
	status := "running"
	cloudNetworkType := "default"
	vcpuType := "shared"

	vm := &emmaSdk.Vm{
		Id:               &id,
		Name:             &name,
		VCpu:             &vcpu,
		RamGb:            &ramGb,
		Status:           &status,
		CloudNetworkType: &cloudNetworkType,
		VCpuType:         &vcpuType,
	}

	// Set default disk
	diskId := int32(1)
	diskSizeGb := int32(50)
	diskTypeId := int32(1)
	diskType := "ssd"
	diskIsBootable := true
	disk := emmaSdk.VmDisksInner{
		Id:         &diskId,
		SizeGb:     &diskSizeGb,
		TypeId:     &diskTypeId,
		Type:       &diskType,
		IsBootable: &diskIsBootable,
	}
	vm.Disks = []emmaSdk.VmDisksInner{disk}

	// Set default network
	networkId := int32(1)
	networkIp := "10.0.0.1"
	networkTypeId := int32(1)
	networkType := "default"
	network := emmaSdk.VmNetworksInner{
		Id:            &networkId,
		Ip:            &networkIp,
		NetworkTypeId: &networkTypeId,
		NetworkType:   &networkType,
	}
	vm.Networks = []emmaSdk.VmNetworksInner{network}

	return &MockVmBuilder{vm: vm}
}

// WithId sets the VM ID
func (b *MockVmBuilder) WithId(id int32) *MockVmBuilder {
	b.vm.Id = &id
	return b
}

// WithName sets the VM name
func (b *MockVmBuilder) WithName(name string) *MockVmBuilder {
	b.vm.Name = &name
	return b
}

// WithVCpu sets the vCPU count
func (b *MockVmBuilder) WithVCpu(vcpu int32) *MockVmBuilder {
	b.vm.VCpu = &vcpu
	return b
}

// WithRamGb sets the RAM size
func (b *MockVmBuilder) WithRamGb(ramGb int32) *MockVmBuilder {
	b.vm.RamGb = &ramGb
	return b
}

// WithStatus sets the VM status
func (b *MockVmBuilder) WithStatus(status string) *MockVmBuilder {
	b.vm.Status = &status
	return b
}

// Build returns the built VM
func (b *MockVmBuilder) Build() *emmaSdk.Vm {
	return b.vm
}

// MockSshKeyBuilder builds mock SSH key data for testing
type MockSshKeyBuilder struct {
	sshKey *emmaSdk.SshKey
}

// NewMockSshKeyBuilder creates a new mock SSH key builder with defaults
func NewMockSshKeyBuilder() *MockSshKeyBuilder {
	id := int32(11111)
	name := "test-ssh-key"
	key := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"
	fingerprint := "SHA256:abcdef1234567890"
	keyType := "RSA"

	sshKey := &emmaSdk.SshKey{
		Id:          &id,
		Name:        &name,
		Key:         &key,
		Fingerprint: &fingerprint,
		KeyType:     &keyType,
	}

	return &MockSshKeyBuilder{sshKey: sshKey}
}

// WithId sets the SSH key ID
func (b *MockSshKeyBuilder) WithId(id int32) *MockSshKeyBuilder {
	b.sshKey.Id = &id
	return b
}

// WithName sets the SSH key name
func (b *MockSshKeyBuilder) WithName(name string) *MockSshKeyBuilder {
	b.sshKey.Name = &name
	return b
}

// WithKey sets the SSH key value
func (b *MockSshKeyBuilder) WithKey(key string) *MockSshKeyBuilder {
	b.sshKey.Key = &key
	return b
}

// WithFingerprint sets the SSH key fingerprint
func (b *MockSshKeyBuilder) WithFingerprint(fingerprint string) *MockSshKeyBuilder {
	b.sshKey.Fingerprint = &fingerprint
	return b
}

// Build returns the built SSH key
func (b *MockSshKeyBuilder) Build() *emmaSdk.SshKey {
	return b.sshKey
}

// MockSecurityGroupBuilder builds mock security group data for testing
type MockSecurityGroupBuilder struct {
	securityGroup *emmaSdk.SecurityGroup
}

// NewMockSecurityGroupBuilder creates a new mock security group builder with defaults
func NewMockSecurityGroupBuilder() *MockSecurityGroupBuilder {
	id := int32(22222)
	name := "test-security-group"
	synchronizationStatus := "Synchronized"
	recomposingStatus := "Idle"

	securityGroup := &emmaSdk.SecurityGroup{
		Id:                    &id,
		Name:                  &name,
		SynchronizationStatus: &synchronizationStatus,
		RecomposingStatus:     &recomposingStatus,
	}

	// Set default rules
	rule1Direction := "inbound"
	rule1Protocol := "tcp"
	rule1Ports := "80,443"
	rule1IpRange := "0.0.0.0/0"
	rule1IsMutable := true
	rule1 := emmaSdk.SecurityGroupRule{
		Direction: &rule1Direction,
		Protocol:  &rule1Protocol,
		Ports:     &rule1Ports,
		IpRange:   &rule1IpRange,
		IsMutable: &rule1IsMutable,
	}

	securityGroup.Rules = []emmaSdk.SecurityGroupRule{rule1}

	return &MockSecurityGroupBuilder{securityGroup: securityGroup}
}

// WithId sets the security group ID
func (b *MockSecurityGroupBuilder) WithId(id int32) *MockSecurityGroupBuilder {
	b.securityGroup.Id = &id
	return b
}

// WithName sets the security group name
func (b *MockSecurityGroupBuilder) WithName(name string) *MockSecurityGroupBuilder {
	b.securityGroup.Name = &name
	return b
}

// WithSynchronizationStatus sets the synchronization status
func (b *MockSecurityGroupBuilder) WithSynchronizationStatus(status string) *MockSecurityGroupBuilder {
	b.securityGroup.SynchronizationStatus = &status
	return b
}

// WithRecomposingStatus sets the recomposing status
func (b *MockSecurityGroupBuilder) WithRecomposingStatus(status string) *MockSecurityGroupBuilder {
	b.securityGroup.RecomposingStatus = &status
	return b
}

// Build returns the built security group
func (b *MockSecurityGroupBuilder) Build() *emmaSdk.SecurityGroup {
	return b.securityGroup
}
