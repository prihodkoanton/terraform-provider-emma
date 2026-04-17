# Error Handling Guide

This guide helps you understand and resolve common errors when using the Emma Terraform Provider.

## Understanding Error Messages

The Emma provider uses a consistent error message format to help you quickly identify and resolve issues:

```
[Operation] [ResourceType] failed: [User-friendly message]

Details:
- Resource ID: [id]
- Status Code: [HTTP status code]
- API Error: [original API error message]
```

### Error Message Components

- **Operation**: The action being performed (Create, Read, Update, Delete, Import)
- **ResourceType**: The type of resource (emma_vm, emma_volume, emma_ssh_key, etc.)
- **User-friendly message**: A clear explanation of what went wrong
- **Resource ID**: The identifier of the resource (when available)
- **Status Code**: HTTP status code from the Emma API
- **API Error**: The original error message from the Emma API

## Common Errors and Solutions

### Authentication Errors

#### Error: "Authentication failed. Please check your credentials."

**Status Code**: 401

**Cause**: Invalid or expired client credentials.

**Solutions**:
1. Verify your `client_id` and `client_secret` are correct
2. Check that credentials haven't expired in the Emma console
3. Ensure environment variables are set correctly if using them:
   ```bash
   export EMMA_CLIENT_ID="your-client-id"
   export EMMA_CLIENT_SECRET="your-client-secret"
   ```
4. Regenerate credentials in the Emma Service application if needed

#### Error: "Permission denied. You don't have access to this resource."

**Status Code**: 403

**Cause**: Your credentials don't have sufficient permissions for the requested operation.

**Solutions**:
1. Check your Emma project permissions
2. Verify the Service application has the required roles
3. Contact your Emma administrator to grant necessary permissions

### Resource Not Found Errors

#### Error: "Resource not found. It may have been deleted."

**Status Code**: 404

**Cause**: The resource doesn't exist in Emma's infrastructure.

**Solutions**:
1. If this occurs during `terraform plan` or `terraform apply`:
   - The resource was deleted outside of Terraform
   - Run `terraform refresh` to sync state
   - Terraform will automatically remove it from state on next read

2. If this occurs during resource creation:
   - Check that referenced resources (data_center_id, os_id, etc.) exist
   - Verify IDs are correct in your configuration

3. If this occurs during import:
   - Verify the resource ID is correct
   - Check that the resource exists in the Emma console

### Validation Errors

#### Error: "Invalid request: [validation details]"

**Status Code**: 400

**Cause**: The request contains invalid data.

**Solutions**:
1. Review the validation details in the error message
2. Check that all required attributes are provided
3. Verify attribute values meet constraints:
   - Volume size within allowed range
   - Valid instance types for the selected provider
   - Proper format for names and identifiers
4. Review the resource documentation for attribute requirements

#### Error: "Validation error: [specific field error]"

**Status Code**: 422

**Cause**: One or more fields failed validation.

**Solutions**:
1. Check the specific field mentioned in the error
2. Common validation issues:
   - **Volume size**: Must be within provider limits (typically 1-10000 GB)
   - **SSH key format**: Must be valid OpenSSH public key format
   - **Security group rules**: Port ranges must be valid (1-65535)
   - **VM configuration**: vCPU and RAM must match available instance types
3. Refer to the Emma API documentation for field constraints

### Resource Conflict Errors

#### Error: "Resource conflict: [conflict details]"

**Status Code**: 409

**Cause**: The operation conflicts with the current state of the resource.

**Solutions**:
1. **Name conflicts**: Resource names must be unique within your project
   - Choose a different name
   - Delete the existing resource if no longer needed

2. **State conflicts**: Resource is in a state that doesn't allow the operation
   - The provider automatically waits for resources to reach stable states
   - The provider automatically retries operations that fail due to state conflicts
   - Check resource status in Emma console if issues persist
   - For VMs: ensure VM is in "running" or "stopped" state before modifications

3. **Attachment conflicts**: Resource is already attached or in use
   - Detach volume before deleting
   - Remove security group associations before deletion

#### Error: "Cannot perform [operation] on [resource]: resource is in [state] state"

**Status Code**: 409

**Cause**: The resource is in a transitional state (BUSY, RECOMPOSING, DRAFT) that doesn't allow the requested operation.

**Solutions**:
1. **Automatic handling**: The provider automatically waits for resources to reach stable states
   - VMs: Waits for POWERED_ON or POWERED_OFF state
   - Volumes: Waits for AVAILABLE state
   - Security Groups: Waits for RECOMPOSED state

2. **If operation times out**:
   - Check resource status in Emma console
   - Increase timeout configuration (see Timeout Configuration section)
   - Verify operation isn't stuck in Emma

3. **Manual intervention**:
   - If resource is stuck in transitional state, check Emma console for errors
   - Contact Emma support if resource remains in transitional state

### Rate Limiting Errors

#### Error: "Rate limit exceeded. Please try again later."

**Status Code**: 429

**Cause**: Too many requests sent to the Emma API in a short time.

**Solutions**:
1. The provider automatically retries with exponential backoff
2. If errors persist:
   - Reduce the number of resources in a single apply
   - Add delays between operations using `time_sleep` resource
   - Contact Emma support to increase rate limits if needed

### Server Errors

#### Error: "Server error. Please try again or contact support."

**Status Code**: 500

**Cause**: Internal server error in the Emma API.

**Solutions**:
1. The provider automatically retries these errors
2. If the error persists after retries:
   - Check Emma status page for service incidents
   - Wait a few minutes and try again
   - Contact Emma support if the issue continues

#### Error: "Service temporarily unavailable. Please try again later."

**Status Code**: 503

**Cause**: Emma service is temporarily unavailable (maintenance, high load, etc.).

**Solutions**:
1. The provider automatically retries with exponential backoff
2. Check Emma status page for scheduled maintenance
3. Wait for service to recover and retry

### Timeout Errors

#### Error: "Timeout waiting for operation to complete"

**Cause**: An asynchronous operation didn't complete within the configured timeout.

**Solutions**:
1. Increase timeout configuration (see Timeout Configuration section below)
2. Check resource status in Emma console to see if operation is still in progress
3. For large resources (VMs with many volumes, large volume resizes):
   - These operations may take longer than default timeouts
   - Increase timeouts in provider configuration
4. If operation failed in Emma:
   - Check Emma console for error details
   - Manually clean up failed resources if needed

#### Error: "Timeout waiting for [resource] to reach state [state]"

**Cause**: A resource didn't reach the expected stable state within the configured timeout.

**Solutions**:
1. **Check resource status**: Look at the Emma console to see current state
2. **Increase timeout**: Use the `timeouts` block in your resource configuration:
   ```hcl
   resource "emma_vm" "example" {
     # ... configuration ...
     
     timeouts {
       create = "45m"  # Increase from default 30m
       update = "30m"
       delete = "15m"
     }
   }
   ```
3. **Check for stuck resources**: If resource is stuck in transitional state:
   - Check Emma console for error messages
   - Contact Emma support if resource won't transition
4. **Verify operation completed**: Sometimes operations complete but status doesn't update:
   - Run `terraform refresh` to sync state
   - Check if resource is actually in desired state

## Retry Behavior

The Emma provider automatically retries failed operations in certain situations to handle transient errors gracefully.

### When Retries Occur

The provider retries operations for:
- **Rate limiting errors** (429): Retries with exponential backoff
- **Server errors** (500, 503): Retries with exponential backoff
- **Network errors**: Temporary connection issues
- **State conflict errors** (409): When resource is in transitional state
  - Automatically waits for resource to reach stable state
  - Retries operation after state transition completes

### State Conflict Retry Behavior

When a resource is in a transitional state (BUSY, RECOMPOSING, DRAFT), the provider:

1. **Detects the state conflict**: Identifies 409 errors or state-related error messages
2. **Waits for stable state**: Polls resource status until it reaches a stable state
3. **Retries the operation**: Attempts the operation again after state transition
4. **Respects timeouts**: Gives up if resource doesn't reach stable state within timeout

Example flow:
```
1. Attempt to attach volume to VM
2. VM is in BUSY state → 409 error
3. Provider waits for VM to reach POWERED_ON state
4. VM reaches POWERED_ON after 30 seconds
5. Provider retries volume attachment
6. Attachment succeeds
```

### When Retries Don't Occur

The provider does NOT retry for:
- **Client errors** (400, 401, 403, 404, 422): These indicate configuration issues
- **Validation errors**: Fix your configuration instead
- **Authentication errors**: Check your credentials
- **Permanent state conflicts**: When resource is in an error state

### Retry Configuration

Default retry settings:
- **Max attempts**: 3
- **Initial delay**: 1 second
- **Max delay**: 30 seconds
- **Backoff multiplier**: 2.0 (exponential)

You can configure retry behavior in the provider configuration:

```hcl
provider "emma" {
  client_id     = var.client_id
  client_secret = var.client_secret
  
  # Retry configuration
  max_retries      = 5              # Maximum retry attempts (default: 3)
  retry_delay      = "2s"           # Initial retry delay (default: 1s)
  max_retry_delay  = "60s"          # Maximum retry delay (default: 30s)
}
```

### Retry Behavior Example

When a rate limit error occurs:
1. **Attempt 1**: Immediate request → 429 error
2. **Attempt 2**: Wait 1 second → retry
3. **Attempt 3**: Wait 2 seconds → retry
4. **Attempt 4**: Wait 4 seconds → retry
5. If still failing, return error to user

## Timeout Configuration

Asynchronous operations (VM creation, volume resizing, security group synchronization) have configurable timeouts.

### Default Timeouts

- **VM operations**: 30 minutes (create/update/delete)
- **Volume operations**: 20 minutes (create/update/delete)
- **Security group operations**: 10 minutes (create/update/delete)
- **Kubernetes cluster operations**: 45 minutes (create/update/delete)
- **State polling**: 10 minutes (waiting for resource to reach stable state)

### Configuring Timeouts

Use Terraform's `timeouts` block in your resource configuration:

```hcl
resource "emma_vm" "example" {
  name               = "my-vm"
  data_center_id     = data.emma_data_center.aws.id
  os_id              = data.emma_operating_system.ubuntu.id
  cloud_network_type = "multi-cloud"
  vcpu               = 2
  ram_gb             = 4
  volume_type        = "ssd"
  volume_gb          = 20
  
  timeouts {
    create = "45m"  # Increase create timeout to 45 minutes
    update = "30m"  # Increase update timeout to 30 minutes
    delete = "15m"  # Increase delete timeout to 15 minutes
  }
}
```

### Timeout Configuration for State Transitions

The provider uses the configured timeouts for both the operation itself and for waiting for resources to reach stable states:

```hcl
resource "emma_volume" "data" {
  name           = "data-volume"
  data_center_id = data.emma_data_center.aws.id
  volume_gb      = 1000
  volume_type    = "ssd"
  
  timeouts {
    create = "30m"  # Used for both volume creation AND waiting for AVAILABLE state
    update = "20m"  # Used for resize AND waiting for state transitions
  }
}
```

### When to Increase Timeouts

Consider increasing timeouts for:
- **Large VMs**: More vCPUs and RAM take longer to provision
- **Large volumes**: Volumes over 1TB may take longer to create
- **Complex configurations**: VMs with many attached volumes
- **Slow providers**: Some cloud providers are slower than others
- **Peak times**: Operations may take longer during high-demand periods
- **State transitions**: Resources that frequently enter transitional states

## Debugging Errors

### Enable Debug Logging

For detailed error information, enable debug logging:

```bash
export TF_LOG=DEBUG
terraform apply
```

This provides:
- Full API request and response details
- Retry attempts and delays
- State transition information
- Detailed error context

See the [Logging Configuration Guide](./LOGGING.md) for more details.

### Check Resource Status

If an operation fails, check the resource status in the Emma console:
1. Log into Emma console
2. Navigate to the resource type (VMs, Volumes, etc.)
3. Check the status and any error messages
4. Look for resources in "error" or "failed" states

### Inspect Terraform State

Check what Terraform knows about the resource:

```bash
# List all resources in state
terraform state list

# Show details of a specific resource
terraform state show emma_vm.example

# Refresh state from Emma API
terraform refresh
```

### Common Debugging Steps

1. **Enable debug logging** to see full error details
2. **Check Emma console** for resource status
3. **Verify credentials** are valid and have correct permissions
4. **Review configuration** for validation errors
5. **Check for conflicts** with existing resources
6. **Inspect state** to ensure it's in sync with Emma
7. **Try manual operation** in Emma console to isolate provider issues

## Recovery Procedures

### Recovering from Failed Operations

#### Failed Create Operation

If resource creation fails:

```bash
# Remove the failed resource from state
terraform state rm emma_vm.example

# Fix the configuration issue
# Edit your .tf files

# Try creating again
terraform apply
```

#### Failed Update Operation

If resource update fails:

```bash
# Refresh state to get current status
terraform refresh

# Check if resource is in a usable state
terraform state show emma_vm.example

# If resource is corrupted, consider recreating:
terraform taint emma_vm.example
terraform apply
```

#### Failed Delete Operation

If resource deletion fails:

```bash
# Try deleting again
terraform destroy -target=emma_vm.example

# If still failing, manually delete in Emma console
# Then remove from state:
terraform state rm emma_vm.example
```

### Handling Partial Failures

When a multi-step operation fails partway through:

1. **Check what succeeded**: Review debug logs and Emma console
2. **Identify the failure point**: Determine which step failed
3. **Manual cleanup if needed**: Remove partially created resources
4. **Update state**: Use `terraform state rm` for resources not in Emma
5. **Fix the issue**: Update configuration or resolve the error
6. **Retry**: Run `terraform apply` again

### State Drift

If Terraform state doesn't match Emma infrastructure:

```bash
# Refresh state from Emma API
terraform refresh

# Review differences
terraform plan

# If resources were deleted outside Terraform:
# They'll be automatically removed from state on next read

# If resources were modified outside Terraform:
# Terraform will show changes to bring them back to desired state
```

## Getting Help

If you continue to experience errors:

1. **Check documentation**: Review resource-specific documentation
2. **Enable debug logging**: Capture full error details
3. **Search issues**: Check GitHub issues for similar problems
4. **Emma support**: Contact Emma support with:
   - Error message and status code
   - Debug logs (with sensitive data removed)
   - Resource configuration
   - Steps to reproduce

## Related Documentation

- [Troubleshooting Guide](./TROUBLESHOOTING.md) - Common issues and fixes
- [Logging Configuration](./LOGGING.md) - Detailed logging setup
- [API Reference](./API_REFERENCE.md) - Provider utilities and functions
- [Architecture Guide](./ARCHITECTURE.md) - Provider design and patterns
