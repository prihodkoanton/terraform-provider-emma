package state

// VM states
var (
	VMStableStates       = []string{"running", "stopped", "POWERED_ON", "POWERED_OFF"}
	VMTransitionalStates = []string{"BUSY", "pending", "starting", "stopping"}
	VMFailureStates      = []string{"error", "failed"}
)

// Volume states
var (
	VolumeStableStates       = []string{"available", "in-use", "AVAILABLE"}
	VolumeTransitionalStates = []string{"BUSY", "DRAFT", "creating", "attaching", "detaching"}
	VolumeFailureStates      = []string{"error", "failed"}
)

// Security Group states
var (
	SecurityGroupStableStates       = []string{"RECOMPOSED"}
	SecurityGroupTransitionalStates = []string{"RECOMPOSING"}
	SecurityGroupFailureStates      = []string{"error", "failed"}
)

// Subnetwork states
var (
	SubnetworkStableStates       = []string{"active"}
	SubnetworkTransitionalStates = []string{"BUSY", "draft"}
	SubnetworkFailureStates      = []string{"error", "failed"}
)

// GetResourceStates returns state definitions for a resource type
func GetResourceStates(resourceType string) (stable, transitional, failure []string) {
	switch resourceType {
	case "vm":
		return VMStableStates, VMTransitionalStates, VMFailureStates
	case "volume":
		return VolumeStableStates, VolumeTransitionalStates, VolumeFailureStates
	case "security_group":
		return SecurityGroupStableStates, SecurityGroupTransitionalStates, SecurityGroupFailureStates
	case "subnetwork":
		return SubnetworkStableStates, SubnetworkTransitionalStates, SubnetworkFailureStates
	default:
		return []string{}, []string{}, []string{}
	}
}
