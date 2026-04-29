package fixtures

import (
	emmaSdk "github.com/emma-community/emma-go-sdk"
)

// VolumeFixture provides test data for volumes
func VolumeFixture() *emmaSdk.Volume {
	id := int32(12345)
	name := "test-volume"
	sizeGb := int32(100)
	volumeType := "ssd"
	isSystem := false
	status := "available"
	projectId := int32(1)
	createdAt := "2025-02-03T10:00:00Z"
	dataCenterId := "dc-test-1"

	volume := &emmaSdk.Volume{}
	volume.SetId(id)
	volume.SetName(name)
	volume.SetSizeGb(sizeGb)
	volume.SetType(volumeType)
	volume.SetIsSystem(isSystem)
	volume.SetStatus(status)
	volume.SetProjectId(projectId)
	volume.SetCreatedAt(createdAt)

	// Set provider info
	provider := emmaSdk.NewVolumeProvider()
	provider.SetId(1)
	provider.SetName("AWS")
	volume.SetProvider(*provider)

	// Set location info
	location := emmaSdk.NewVolumeLocation()
	location.SetId(1)
	location.SetName("US East")
	location.SetContinent("North America")
	location.SetRegion("us-east-1")
	volume.SetLocation(*location)

	// Set data center info
	dataCenter := emmaSdk.NewVolumeDataCenter()
	dataCenter.SetId(dataCenterId)
	dataCenter.SetName("Test Data Center")
	volume.SetDataCenter(*dataCenter)

	return volume
}

// VmFixture provides test data for VMs
func VmFixture() *emmaSdk.Vm {
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

	// Set disks
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

	// Set networks
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

	// Set cost
	costUnit := "month"
	costCurrency := "USD"
	costPrice := float32(50.0)
	cost := emmaSdk.VmCost{
		Unit:     &costUnit,
		Currency: &costCurrency,
		Price:    &costPrice,
	}
	vm.Cost = &cost

	return vm
}

// SshKeyFixture provides test data for SSH keys
func SshKeyFixture() *emmaSdk.SshKey {
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

	return sshKey
}

// SecurityGroupFixture provides test data for security groups
func SecurityGroupFixture() *emmaSdk.SecurityGroup {
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

	// Set rules
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

	rule2Direction := "outbound"
	rule2Protocol := "all"
	rule2Ports := ""
	rule2IpRange := "0.0.0.0/0"
	rule2IsMutable := true
	rule2 := emmaSdk.SecurityGroupRule{
		Direction: &rule2Direction,
		Protocol:  &rule2Protocol,
		Ports:     &rule2Ports,
		IpRange:   &rule2IpRange,
		IsMutable: &rule2IsMutable,
	}

	securityGroup.Rules = []emmaSdk.SecurityGroupRule{rule1, rule2}

	return securityGroup
}
