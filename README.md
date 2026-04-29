# Terraform Provider Emma

## Overview

This [Terraform Provider Emma](https://registry.terraform.io/providers/emma-community/emma/latest) allows you to manage 
multi-cloud resources. The [emma platform](https://www.emma.ms/) empowers you to effortlessly deploy and manage cloud 
resources across diverse environments, spanning on-premises, private, and public clouds. Whether you're a seasoned cloud 
professional honing your multi-cloud setup or diving into cloud management for the first time, our cloud-agnostic 
approach guarantees freedom to leverage the right cloud services you need.

## Features

### Resource Management
- **Virtual Machines**: Provision and manage VMs across multiple cloud providers
- **Spot Instances**: Utilize spot instances for cost-effective computing
- **Storage Volumes**: Create, attach, resize, and manage storage volumes
- **Kubernetes Clusters**: Deploy and manage Kubernetes clusters
- **SSH Keys**: Manage SSH keys for secure instance access
- **Security Groups**: Define and manage security groups to control network traffic

### Provider Capabilities
- **Multi-Cloud Support**: Deploy resources across on-premises, private, and public clouds
- **Comprehensive Error Handling**: Clear, actionable error messages with automatic retry logic
- **Async Operations**: Reliable handling of long-running operations with configurable timeouts
- **Resource Import**: Import existing Emma resources into Terraform management
- **State Management**: Automatic drift detection and state synchronization
- **Structured Logging**: Debug-friendly logging with sensitive data sanitization
- **Property-Based Testing**: Extensively tested for reliability and correctness

## Installation

1. **Prerequisites**: Ensure you have Terraform installed on your system. You can download it from the [Terraform website](https://developer.hashicorp.com/terraform/install).
2. **Define Provider Configuration**: To install this provider, copy and paste this code into your Terraform configuration. 
Then, run `terraform init`:
   ```hcl
   terraform {
     required_providers {
       emma = {
         source = "emma-community/emma"
         version = "0.1.0"
         }
      }
   }

   provider "emma" {
     client_id     = "your client id"
     client_secret = "your client secret"
   }
   ```

3. **Define Resources**: Define the resources you want to manage in your Terraform configuration. Here's an example 
of provisioning a virtual machine, you can find more documentation on the [terraform provider page](https://registry.terraform.io/providers/emma-community/emma/latest/docs):
   ```hcl
   resource "emma_vm" "vm" {
      name               = "Example"
      data_center_id     = data.emma_data_center.aws.id
      os_id              = data.emma_operating_system.ubuntu.id
      cloud_network_type = "multi-cloud"
      vcpu_type          = "shared"
      vcpu               = 2
      ram_gb             = 1
      volume_type        = "ssd"
      volume_gb          = 8
      ssh_key_id         = emma_ssh_key.ssh_key.id
   }
   ```

4. **Run Terraform Commands**: Use Terraform commands (`terraform plan`, `terraform apply`, etc.) 
to apply your configuration and manage your infrastructure.

## Authentication

To authenticate with Emma's infrastructure, provide the necessary credentials using the `client_id` and `client_secret` 
options in your provider configuration.

### Configuration Options

```hcl
provider "emma" {
  # Required: Authentication credentials
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
  
  # Optional: API endpoint (defaults to https://api.emma.ms)
  host = "https://api.emma.ms"
  
  # Optional: Retry configuration
  max_retries      = 3      # Maximum retry attempts (default: 3)
  retry_delay      = "1s"   # Initial retry delay (default: 1s)
  max_retry_delay  = "30s"  # Maximum retry delay (default: 30s)
}
```

### Environment Variables

Alternatively, use environment variables:

```bash
export EMMA_CLIENT_ID="your-client-id"
export EMMA_CLIENT_SECRET="your-client-secret"
export EMMA_HOST="https://api.emma.ms"  # Optional
```

## Advanced Features

### Resource Import

Import existing Emma resources into Terraform:

```bash
# Import a virtual machine
terraform import emma_vm.example 12345

# Import a volume
terraform import emma_volume.data 67890

# Import an SSH key
terraform import emma_ssh_key.key 11111
```

See individual resource documentation for import syntax.

### Timeout Configuration

Configure timeouts for long-running operations:

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
    create = "45m"  # Increase timeout for large VMs
    update = "30m"
    delete = "15m"
  }
}
```

### Error Handling and Retry

The provider automatically handles transient errors:
- **Rate limiting (429)**: Automatic retry with exponential backoff
- **Server errors (500, 503)**: Automatic retry with exponential backoff
- **Network errors**: Automatic retry for temporary connection issues
- **Client errors (4xx)**: Immediate failure with clear error messages

Configure retry behavior in the provider block (see Authentication section above).

## Debugging and Logging

The provider supports comprehensive logging to help troubleshoot issues.

### Enable Debug Logging

```bash
export TF_LOG=DEBUG
terraform apply
```

### Save Logs to File

```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log
terraform apply
```

### Log Levels

- `TRACE`: Most verbose, includes all details
- `DEBUG`: Detailed information for debugging
- `INFO`: General informational messages
- `WARN`: Warning messages
- `ERROR`: Error messages only

### What Gets Logged

- API requests and responses
- State transitions
- Retry attempts
- Validation results
- Error details with context

**Note**: Sensitive data (passwords, tokens, keys) is automatically sanitized in logs.

For detailed logging configuration, see the [Logging Configuration Guide](./docs/LOGGING.md).

## Documentation

### User Guides
- [Error Handling Guide](./docs/ERROR_HANDLING.md) - Common errors and solutions
- [Troubleshooting Guide](./docs/TROUBLESHOOTING.md) - Debugging and fixes
- [Migration Guide](./docs/MIGRATION.md) - Upgrading between versions
- [Logging Configuration](./docs/LOGGING.md) - Detailed logging setup

### Developer Guides
- [Architecture Guide](./docs/ARCHITECTURE.md) - Provider design and patterns
- [API Reference](./docs/API_REFERENCE.md) - Utilities and functions
- [Testing Guide](./docs/TESTING_GUIDE.md) - Writing and running tests

### Resource Documentation

Complete resource and data source documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/emma-community/emma/latest/docs).

## Examples

Find complete examples in the [examples/](./examples/) directory:

- [Virtual Machines](./examples/resources/emma_vm/)
- [Storage Volumes](./examples/resources/emma_volume/)
- [SSH Keys](./examples/resources/emma_ssh_key/)
- [Security Groups](./examples/resources/emma_security_group/)
- [Spot Instances](./examples/resources/emma_spot_instance/)
- [Kubernetes Clusters](./examples/resources/emma_kubernetes_cluster/)
- [Data Sources](./examples/data-sources/)

## Support

### Versioning

This provider follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version (X.0.0): Incompatible API changes or breaking changes
- **MINOR** version (0.X.0): New features in a backwards-compatible manner
- **PATCH** version (0.0.X): Backwards-compatible bug fixes

**Current Version**: 0.1.0

**Version Compatibility**:
- Terraform >= 1.0
- Emma Go SDK v0.0.10
- Go 1.22.7+

**Release Notes**: See [CHANGELOG.md](./CHANGELOG.md) for detailed release history and [RELEASE_NOTES.md](./RELEASE_NOTES.md) for the latest release information.

### Getting Help

1. **Documentation**: Check the guides above and [Terraform Registry docs](https://registry.terraform.io/providers/emma-community/emma/latest/docs)
2. **Troubleshooting**: See the [Troubleshooting Guide](./docs/TROUBLESHOOTING.md)
3. **Issues**: Report bugs or request features on [GitHub Issues](https://github.com/emma-community/terraform-provider-emma/issues)
4. **Emma Support**: Contact Emma support for API or account issues

### Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

This provider is distributed under the Mozilla Public License 2.0. See [LICENSE](./LICENSE) for details.
