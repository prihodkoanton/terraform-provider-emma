# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the Emma Terraform Provider.

## Table of Contents

- [Enabling Debug Logging](#enabling-debug-logging)
- [Interpreting Error Messages](#interpreting-error-messages)
- [Common Issues and Fixes](#common-issues-and-fixes)
- [Performance Issues](#performance-issues)
- [State Management Issues](#state-management-issues)
- [Advanced Debugging](#advanced-debugging)

## Enabling Debug Logging

Debug logging is your first tool for troubleshooting provider issues. It shows detailed information about API calls, state changes, and internal operations.

### Basic Debug Logging

Enable debug logging for all Terraform operations:

```bash
export TF_LOG=DEBUG
terraform apply
```

### Filtering Logs by Level

Set different log levels for different components:

```bash
# Only show errors
export TF_LOG=ERROR

# Show info and above (INFO, WARN, ERROR)
export TF_LOG=INFO

# Show debug and above (DEBUG, INFO, WARN, ERROR)
export TF_LOG=DEBUG

# Show everything including trace
export TF_LOG=TRACE
```

### Saving Logs to File

Capture logs to a file for analysis:

```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log
terraform apply
```

### Provider-Specific Logging

Enable logging only for the Emma provider:

```bash
export TF_LOG_PROVIDER=DEBUG
terraform apply
```

### What Debug Logs Show

Debug logs include:

1. **API Requests**: Full HTTP requests sent to Emma API
   ```
   [DEBUG] API Request: POST /v1/vms
   [DEBUG] Request Body: {"name":"my-vm","vcpu":2,...}
   ```

2. **API Responses**: Full HTTP responses from Emma API
   ```
   [DEBUG] API Response: 200 OK
   [DEBUG] Response Body: {"id":12345,"status":"creating",...}
   ```

3. **State Transitions**: Changes to Terraform state
   ```
   [INFO] Resource created: emma_vm.example (ID: 12345)
   [INFO] State updated: emma_vm.example
   ```

4. **Retry Attempts**: When operations are retried
   ```
   [DEBUG] Retry attempt 1/3 after 1s delay
   [DEBUG] Retry attempt 2/3 after 2s delay
   ```

5. **Validation**: Input validation results
   ```
   [DEBUG] Validating attribute: volume_gb
   [DEBUG] Validation passed: volume_gb=100
   ```

### Sensitive Data in Logs

The provider automatically sanitizes sensitive data in logs:
- Passwords are replaced with `[REDACTED]`
- Tokens are replaced with `[REDACTED]`
- SSH private keys are replaced with `[REDACTED]`

However, always review logs before sharing to ensure no sensitive data is exposed.

## Interpreting Error Messages

### Error Message Structure

Emma provider errors follow a consistent format:

```
[Operation] [ResourceType] failed: [User-friendly message]

Details:
- Resource ID: 12345
- Status Code: 400
- API Error: Invalid volume size: must be between 1 and 10000
```

### Key Components

1. **Operation**: What was being attempted
   - `Create`: Creating a new resource
   - `Read`: Reading resource state
   - `Update`: Modifying an existing resource
   - `Delete`: Removing a resource
   - `Import`: Importing existing resource

2. **ResourceType**: Which resource type
   - `emma_vm`: Virtual machine
   - `emma_volume`: Storage volume
   - `emma_ssh_key`: SSH key
   - `emma_security_group`: Security group
   - `emma_spot_instance`: Spot instance
   - `emma_kubernetes_cluster`: Kubernetes cluster

3. **Status Code**: HTTP status code
   - `400`: Bad request (configuration error)
   - `401`: Authentication failed
   - `403`: Permission denied
   - `404`: Resource not found
   - `409`: Resource conflict
   - `422`: Validation error
   - `429`: Rate limit exceeded
   - `500`: Server error
   - `503`: Service unavailable

4. **API Error**: Original error from Emma API
   - Provides specific details about what went wrong
   - May include field names and validation constraints

### Reading Stack Traces

When errors occur, Terraform may show a stack trace:

```
Error: Create emma_vm failed: Invalid request

  on main.tf line 10, in resource "emma_vm" "example":
  10: resource "emma_vm" "example" {

Details:
- Status Code: 400
- API Error: vcpu must be a positive integer
```

Key information:
- **File and line**: Where the error occurred in your configuration
- **Resource block**: Which resource definition has the issue
- **Error details**: Specific information about the failure

## Common Issues and Fixes

### Issue: "Authentication failed"

**Symptoms**:
- Error message: "Authentication failed. Please check your credentials."
- Status code: 401
- Occurs on any operation

**Diagnosis**:
```bash
# Check if credentials are set
echo $EMMA_CLIENT_ID
echo $EMMA_CLIENT_SECRET

# Test authentication manually
curl -X POST https://api.emma.ms/v1/auth/token \
  -d "client_id=$EMMA_CLIENT_ID" \
  -d "client_secret=$EMMA_CLIENT_SECRET"
```

**Solutions**:
1. Verify credentials in provider configuration:
   ```hcl
   provider "emma" {
     client_id     = "your-client-id"
     client_secret = "your-client-secret"
   }
   ```

2. Check environment variables:
   ```bash
   export EMMA_CLIENT_ID="your-client-id"
   export EMMA_CLIENT_SECRET="your-client-secret"
   ```

3. Regenerate credentials in Emma console if expired

### Issue: "Resource not found" during plan/apply

**Symptoms**:
- Error message: "Resource not found. It may have been deleted."
- Status code: 404
- Resource was previously working

**Diagnosis**:
```bash
# Check if resource exists in Emma console
# Check Terraform state
terraform state show emma_vm.example

# Enable debug logging to see API calls
export TF_LOG=DEBUG
terraform plan
```

**Solutions**:
1. Resource was deleted outside Terraform:
   ```bash
   # Refresh state to sync with Emma
   terraform refresh
   
   # Terraform will remove it from state automatically
   terraform plan
   ```

2. Resource ID changed:
   - Check Emma console for correct ID
   - Update configuration or re-import resource

3. Wrong project/environment:
   - Verify credentials point to correct Emma project

### Issue: Resource stuck in transitional state

**Symptoms**:
- Resource remains in BUSY, RECOMPOSING, or DRAFT state
- Operations timeout waiting for stable state
- Provider logs show repeated state checks

**Diagnosis**:
```bash
# Enable debug logging to see state polling
export TF_LOG=DEBUG
terraform apply

# Look for state check messages:
# [DEBUG] Checking resource state: BUSY
# [DEBUG] Resource still in transitional state, waiting...
# [DEBUG] Checking resource state: BUSY
```

**Solutions**:
1. **Wait for completion**: The provider automatically polls until stable state
   - Check Emma console to verify operation is progressing
   - Look for any error messages in Emma console

2. **Increase timeout**: If operation is legitimately slow:
   ```hcl
   resource "emma_vm" "example" {
     # ... configuration ...
     
     timeouts {
       update = "45m"  # Increase from default 30m
     }
   }
   ```

3. **Check for stuck operations**: If resource won't transition:
   - Check Emma console for error state
   - Contact Emma support if resource is stuck
   - May need to manually intervene in Emma console

4. **Verify operation completed**: Sometimes status doesn't update:
   - Run `terraform refresh` to sync state
   - Check if operation actually completed in Emma console

### Issue: "Timeout waiting for operation to complete"

**Symptoms**:
- Operation starts but never completes
- Error after several minutes
- Resource may be in "creating" or "updating" state

**Diagnosis**:
```bash
# Check resource status in Emma console
# Enable debug logging to see polling attempts
export TF_LOG=DEBUG
terraform apply

# Check if operation is still running
# Look for status checks in logs:
# [DEBUG] Checking status: creating
# [DEBUG] Checking status: creating
# ...
```

**Solutions**:
1. Increase timeout in resource configuration:
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

2. Check Emma console for operation status:
   - If operation completed: Run `terraform refresh`
   - If operation failed: Check Emma error message
   - If operation stuck: Contact Emma support

3. For large resources, use longer timeouts:
   ```hcl
   resource "emma_volume" "large" {
     volume_gb = 5000  # Large volume
     
     timeouts {
       create = "60m"  # Large volumes take longer
     }
   }
   ```

### Issue: "Rate limit exceeded"

**Symptoms**:
- Error message: "Rate limit exceeded. Please try again later."
- Status code: 429
- Occurs during large deployments

**Diagnosis**:
```bash
# Check how many resources are being created
terraform plan | grep "will be created"

# Enable debug logging to see retry attempts
export TF_LOG=DEBUG
terraform apply
```

**Solutions**:
1. Provider automatically retries with backoff - wait for completion

2. Reduce concurrent operations:
   ```bash
   # Apply resources in smaller batches
   terraform apply -target=emma_vm.vm1
   terraform apply -target=emma_vm.vm2
   ```

3. Increase retry configuration:
   ```hcl
   provider "emma" {
     max_retries     = 5   # Increase from default 3
     max_retry_delay = "60s"  # Increase max delay
   }
   ```

4. Add delays between resources:
   ```hcl
   resource "time_sleep" "wait" {
     create_duration = "10s"
   }
   
   resource "emma_vm" "vm2" {
     depends_on = [time_sleep.wait]
     # ... configuration ...
   }
   ```

### Issue: "Validation error" on valid configuration

**Symptoms**:
- Error message: "Validation error: [field] is invalid"
- Status code: 422
- Configuration looks correct

**Diagnosis**:
```bash
# Enable debug logging to see validation details
export TF_LOG=DEBUG
terraform plan

# Check provider documentation for field constraints
terraform providers schema -json | jq '.provider_schemas'
```

**Solutions**:
1. Check field constraints in documentation:
   - Volume size limits vary by provider
   - Instance types vary by data center
   - Some fields are mutually exclusive

2. Verify referenced resources exist:
   ```hcl
   # Make sure data sources return valid IDs
   data "emma_data_center" "aws" {
     name = "AWS US East"
   }
   
   output "dc_id" {
     value = data.emma_data_center.aws.id
   }
   ```

3. Check for typos in enum values:
   ```hcl
   # Correct
   cloud_network_type = "multi-cloud"
   
   # Incorrect
   cloud_network_type = "multicloud"  # Missing hyphen
   ```

### Issue: State drift detected

**Symptoms**:
- `terraform plan` shows changes when none were made
- Resources modified outside Terraform
- Unexpected updates proposed

**Diagnosis**:
```bash
# See what changed
terraform plan

# Show current state
terraform state show emma_vm.example

# Refresh state from Emma
terraform refresh

# Compare with Emma console
```

**Solutions**:
1. Accept drift and update state:
   ```bash
   terraform refresh
   terraform apply
   ```

2. Revert changes in Emma console to match Terraform

3. Update Terraform configuration to match current state

4. Use lifecycle rules to ignore certain attributes:
   ```hcl
   resource "emma_vm" "example" {
     # ... configuration ...
     
     lifecycle {
       ignore_changes = [
         tags,  # Ignore tag changes made outside Terraform
       ]
     }
   }
   ```

### Issue: "Resource conflict" errors

**Symptoms**:
- Error message: "Resource conflict: [details]"
- Status code: 409
- Resource name already exists or resource in wrong state

**Diagnosis**:
```bash
# Check for existing resources with same name
# in Emma console

# Check resource state
terraform state show emma_vm.example

# Enable debug logging
export TF_LOG=DEBUG
terraform apply
```

**Solutions**:
1. Name conflicts:
   ```hcl
   # Use unique names
   resource "emma_vm" "example" {
     name = "my-vm-${terraform.workspace}"  # Add workspace suffix
   }
   ```

2. State conflicts (automatic handling):
   - The provider automatically waits for resources to reach stable states
   - The provider automatically retries operations after state transitions
   - Check debug logs to see state polling progress:
     ```
     [DEBUG] Waiting for VM to reach stable state
     [DEBUG] Current state: BUSY, target states: [POWERED_ON, POWERED_OFF]
     [DEBUG] VM reached stable state: POWERED_ON
     ```

3. If automatic retry fails:
   - Check Emma console for resource status
   - Verify resource isn't stuck in error state
   - Increase timeout if needed:
     ```hcl
     resource "emma_vm" "example" {
       # ... configuration ...
       
       timeouts {
         update = "30m"  # Increase timeout for state transitions
       }
     }
     ```

4. Attachment conflicts:
   ```hcl
   # Detach before deleting
   resource "emma_volume" "data" {
     # ... configuration ...
   }
   
   # Remove attachment first
   # Then delete volume
   ```

### Issue: Import fails with "Resource not found"

**Symptoms**:
- `terraform import` command fails
- Error: "Resource not found"
- Resource exists in Emma console

**Diagnosis**:
```bash
# Verify resource ID
# Check Emma console for correct ID

# Test API access
export TF_LOG=DEBUG
terraform import emma_vm.example 12345
```

**Solutions**:
1. Verify resource ID format:
   ```bash
   # Correct: numeric ID
   terraform import emma_vm.example 12345
   
   # Incorrect: name instead of ID
   terraform import emma_vm.example "my-vm"
   ```

2. Check resource type matches:
   ```bash
   # Make sure resource type in config matches import
   # Config must exist before import:
   resource "emma_vm" "example" {
     # Minimal config for import
   }
   ```

3. Verify credentials have read access to resource

## Performance Issues

### Issue: Slow plan/apply operations

**Symptoms**:
- `terraform plan` takes several minutes
- `terraform apply` is very slow
- Many API calls being made

**Diagnosis**:
```bash
# Enable debug logging to see API calls
export TF_LOG=DEBUG
terraform plan 2>&1 | grep "API Request" | wc -l

# Profile Terraform execution
terraform plan -json | jq '.type'
```

**Solutions**:
1. Reduce unnecessary refreshes:
   ```bash
   # Skip refresh if state is current
   terraform plan -refresh=false
   ```

2. Use targeted operations:
   ```bash
   # Only plan specific resources
   terraform plan -target=emma_vm.example
   ```

3. Optimize data source usage:
   ```hcl
   # Cache data source results
   data "emma_data_center" "aws" {
     name = "AWS US East"
   }
   
   # Reuse across multiple resources
   resource "emma_vm" "vm1" {
     data_center_id = data.emma_data_center.aws.id
   }
   
   resource "emma_vm" "vm2" {
     data_center_id = data.emma_data_center.aws.id
   }
   ```

4. Break large configurations into modules:
   ```hcl
   # Instead of 100 resources in one file
   # Split into modules
   module "compute" {
     source = "./modules/compute"
   }
   
   module "storage" {
     source = "./modules/storage"
   }
   ```

### Issue: High memory usage

**Symptoms**:
- Terraform process uses excessive memory
- System becomes slow during operations
- Out of memory errors

**Solutions**:
1. Reduce parallelism:
   ```bash
   terraform apply -parallelism=5  # Default is 10
   ```

2. Split state into smaller pieces:
   ```hcl
   # Use separate state files for different environments
   terraform {
     backend "s3" {
       key = "prod/terraform.tfstate"
     }
   }
   ```

3. Use remote state for large deployments

## State Management Issues

### Issue: State file corruption

**Symptoms**:
- Error: "Failed to load state"
- State file appears corrupted
- Cannot run any Terraform commands

**Solutions**:
1. Restore from backup:
   ```bash
   # Terraform creates automatic backups
   cp terraform.tfstate.backup terraform.tfstate
   ```

2. Use remote state with versioning:
   ```hcl
   terraform {
     backend "s3" {
       bucket         = "my-terraform-state"
       key            = "emma/terraform.tfstate"
       region         = "us-east-1"
       encrypt        = true
       versioning     = true  # Enable versioning
     }
   }
   ```

3. Manually reconstruct state:
   ```bash
   # Remove corrupted state
   mv terraform.tfstate terraform.tfstate.corrupted
   
   # Import resources one by one
   terraform import emma_vm.vm1 12345
   terraform import emma_vm.vm2 67890
   ```

### Issue: State lock errors

**Symptoms**:
- Error: "Error acquiring the state lock"
- Cannot run Terraform commands
- Lock persists after operation completes

**Solutions**:
1. Wait for lock to release (if operation is running)

2. Force unlock (if operation crashed):
   ```bash
   # Get lock ID from error message
   terraform force-unlock <lock-id>
   ```

3. Check for stuck processes:
   ```bash
   # Find Terraform processes
   ps aux | grep terraform
   
   # Kill stuck processes
   kill <pid>
   ```

### Issue: State out of sync with reality

**Symptoms**:
- Terraform state doesn't match Emma console
- Resources exist in Emma but not in state
- Resources in state but not in Emma

**Solutions**:
1. Refresh state:
   ```bash
   terraform refresh
   ```

2. Import missing resources:
   ```bash
   terraform import emma_vm.example 12345
   ```

3. Remove deleted resources from state:
   ```bash
   terraform state rm emma_vm.deleted
   ```

4. Rebuild state from scratch:
   ```bash
   # Export resource IDs from Emma
   # Import each resource
   terraform import emma_vm.vm1 12345
   terraform import emma_volume.vol1 67890
   ```

## Advanced Debugging

### Capturing HTTP Traffic

Use a proxy to inspect HTTP traffic between Terraform and Emma API:

```bash
# Using mitmproxy
mitmproxy -p 8080

# Configure Terraform to use proxy
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080
terraform apply
```

### Analyzing API Responses

Extract API responses from debug logs:

```bash
# Save debug logs
export TF_LOG=DEBUG
export TF_LOG_PATH=debug.log
terraform apply

# Extract API responses
grep "API Response" debug.log > responses.log

# Parse JSON responses
grep "Response Body" debug.log | sed 's/.*Response Body: //' | jq .
```

### Testing Provider Locally

Build and test provider locally:

```bash
# Build provider
go build -o terraform-provider-emma

# Install locally
mkdir -p ~/.terraform.d/plugins/hashicorp.com/edu/emma/0.1.0/darwin_amd64
cp terraform-provider-emma ~/.terraform.d/plugins/hashicorp.com/edu/emma/0.1.0/darwin_amd64/

# Use local provider
terraform {
  required_providers {
    emma = {
      source = "hashicorp.com/edu/emma"
      version = "0.1.0"
    }
  }
}
```

### Running Provider in Debug Mode

Run provider with debugger attached:

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o terraform-provider-emma

# Run with delve
dlv exec ./terraform-provider-emma -- -debug

# In another terminal, set TF_REATTACH_PROVIDERS
export TF_REATTACH_PROVIDERS='...'  # From dlv output
terraform apply
```

### Checking Provider Version

Verify provider version:

```bash
# Show provider version
terraform version

# Show provider schema
terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/emma-community/emma"'

# List installed providers
terraform providers
```

## Getting Additional Help

If issues persist after troubleshooting:

1. **Gather information**:
   - Debug logs (with sensitive data removed)
   - Terraform version: `terraform version`
   - Provider version: `terraform providers`
   - Configuration files (sanitized)
   - Steps to reproduce

2. **Check existing issues**:
   - GitHub issues: https://github.com/emma-community/terraform-provider-emma/issues
   - Search for similar problems

3. **Create detailed issue report**:
   - Clear description of problem
   - Expected vs actual behavior
   - Debug logs
   - Configuration example
   - Environment details

4. **Contact Emma support**:
   - For API-related issues
   - For account/permission issues
   - For service availability issues

## Related Documentation

- [Error Handling Guide](./ERROR_HANDLING.md) - Common errors and solutions
- [Logging Configuration](./LOGGING.md) - Detailed logging setup
- [API Reference](./API_REFERENCE.md) - Provider utilities and functions
- [Architecture Guide](./ARCHITECTURE.md) - Provider design and patterns
