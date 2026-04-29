# Migration Guide

This guide helps you upgrade the Emma Terraform Provider to newer versions and understand any breaking changes or deprecated features.

## Current Version: 0.1.0

The Emma Terraform Provider is currently in its initial release (0.1.0). This migration guide will be updated as new versions are released with breaking changes or deprecated features.

## Version Compatibility

### Terraform Version Requirements

The Emma provider requires:
- **Terraform**: >= 1.0.0
- **Go**: >= 1.22.7 (for development)

### Tested Terraform Versions

The provider is tested against:
- Terraform 1.0.x
- Terraform 1.1.x
- Terraform 1.2.x
- Terraform 1.3.x
- Terraform 1.4.x
- Terraform 1.5.x
- Terraform 1.6.x
- Terraform 1.7.x
- Terraform 1.8.x

## Semantic Versioning

The Emma provider follows [Semantic Versioning](https://semver.org/):

- **Major version** (X.0.0): Breaking changes that require configuration updates
- **Minor version** (0.X.0): New features, backward compatible
- **Patch version** (0.0.X): Bug fixes, backward compatible

### What Constitutes a Breaking Change

Breaking changes include:
- Removing or renaming resource types
- Removing or renaming resource attributes
- Changing attribute types (e.g., string to number)
- Changing default values that affect existing resources
- Removing or changing provider configuration options
- Changing resource import ID format

### What is NOT a Breaking Change

These changes are backward compatible:
- Adding new resource types
- Adding new optional attributes
- Adding new data sources
- Improving error messages
- Adding validation to catch errors earlier
- Performance improvements
- Bug fixes that correct incorrect behavior

## Future Migration Paths

As the provider evolves, this section will document migration paths between major versions.

### Preparing for Future Upgrades

To make future upgrades easier:

1. **Pin provider version** in your configuration:
   ```hcl
   terraform {
     required_providers {
       emma = {
         source  = "emma-community/emma"
         version = "~> 0.1.0"  # Allow patch updates only
       }
     }
   }
   ```

2. **Use version constraints** appropriately:
   ```hcl
   # Pessimistic constraint: allows 0.1.x updates
   version = "~> 0.1.0"
   
   # Exact version: no automatic updates
   version = "= 0.1.0"
   
   # Range: allows 0.1.x through 0.2.x
   version = ">= 0.1.0, < 0.3.0"
   ```

3. **Test upgrades** in non-production environments first

4. **Review release notes** before upgrading

5. **Keep state backups** before major upgrades

## Upgrade Process

### Standard Upgrade Process

For minor and patch version upgrades:

1. **Review release notes**:
   - Check [CHANGELOG.md](../CHANGELOG.md) for changes
   - Review [GitHub releases](https://github.com/emma-community/terraform-provider-emma/releases)

2. **Update provider version**:
   ```hcl
   terraform {
     required_providers {
       emma = {
         source  = "emma-community/emma"
         version = "~> 0.2.0"  # Update version
       }
     }
   }
   ```

3. **Reinitialize Terraform**:
   ```bash
   terraform init -upgrade
   ```

4. **Test in non-production**:
   ```bash
   terraform plan
   terraform apply
   ```

5. **Verify resources**:
   ```bash
   # Check that resources are still managed correctly
   terraform state list
   terraform show
   ```

6. **Deploy to production** after successful testing

### Major Version Upgrade Process

For major version upgrades (when available):

1. **Backup state**:
   ```bash
   cp terraform.tfstate terraform.tfstate.backup
   # Or use remote state versioning
   ```

2. **Review migration guide** for the specific version

3. **Update configuration** according to breaking changes

4. **Test thoroughly** in non-production environment

5. **Plan upgrade** with `terraform plan`:
   ```bash
   terraform plan -out=upgrade.tfplan
   ```

6. **Review plan carefully** for unexpected changes

7. **Apply upgrade**:
   ```bash
   terraform apply upgrade.tfplan
   ```

8. **Verify all resources** are functioning correctly

## Deprecation Policy

When features are deprecated:

1. **Deprecation notice**: Feature marked as deprecated in documentation
2. **Warning period**: Minimum one major version before removal
3. **Deprecation warnings**: Provider logs warnings when deprecated features are used
4. **Migration path**: Alternative approach documented
5. **Removal**: Feature removed in next major version

### Checking for Deprecated Features

Enable warnings to see if you're using deprecated features:

```bash
export TF_LOG=WARN
terraform plan
```

Look for messages like:
```
[WARN] Deprecated: The 'old_attribute' attribute is deprecated and will be removed in version 1.0.0. Use 'new_attribute' instead.
```

## Known Issues and Workarounds

### Current Known Issues

No known issues in version 0.1.0.

This section will be updated as issues are discovered and workarounds are identified.

## Breaking Changes by Version

### Version 0.1.0 (Initial Release)

No breaking changes - initial release.

## New Features by Version

### Version 0.1.0 (Initial Release)

**Resources**:
- `emma_vm`: Virtual machine management
- `emma_volume`: Storage volume management
- `emma_ssh_key`: SSH key management
- `emma_security_group`: Security group management
- `emma_spot_instance`: Spot instance management
- `emma_kubernetes_cluster`: Kubernetes cluster management

**Data Sources**:
- `emma_data_center`: Query available data centers
- `emma_location`: Query available locations
- `emma_operating_system`: Query available operating systems
- `emma_provider`: Query cloud provider information
- `emma_volume`: Query existing volumes
- `emma_volume_configurations`: Query volume configuration options

**Features**:
- OAuth2 authentication with client credentials
- Comprehensive error handling with retry logic
- Async operation support with configurable timeouts
- Property-based testing for reliability
- Structured logging with sensitive data sanitization
- Resource import support
- Cross-field validation
- State drift detection

## Deprecated Features

### Version 0.1.0

No deprecated features in initial release.

## Configuration Changes

### Provider Configuration

Current provider configuration (0.1.0):

```hcl
provider "emma" {
  # Required
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
  
  # Optional
  host              = "https://api.emma.ms"  # Default
  max_retries       = 3                       # Default
  retry_delay       = "1s"                    # Default
  max_retry_delay   = "30s"                   # Default
}
```

### Environment Variables

Supported environment variables (0.1.0):
- `EMMA_CLIENT_ID`: Client ID for authentication
- `EMMA_CLIENT_SECRET`: Client secret for authentication
- `EMMA_HOST`: Emma API host URL

## Resource Schema Changes

### Version 0.1.0

Initial schemas - no changes yet.

Future versions will document schema changes here.

## State Format Changes

### Version 0.1.0

Initial state format - no changes yet.

The provider uses Terraform Plugin Framework's standard state format. State format changes will be documented here in future versions.

## Testing Your Migration

### Pre-Migration Checklist

Before upgrading:

- [ ] Review release notes and migration guide
- [ ] Backup Terraform state
- [ ] Test upgrade in non-production environment
- [ ] Review `terraform plan` output carefully
- [ ] Verify no unexpected resource replacements
- [ ] Check for deprecation warnings
- [ ] Update CI/CD pipelines if needed

### Post-Migration Checklist

After upgrading:

- [ ] Verify all resources are in expected state
- [ ] Run `terraform plan` to ensure no drift
- [ ] Test CRUD operations on resources
- [ ] Verify import functionality still works
- [ ] Check logs for warnings or errors
- [ ] Update documentation and runbooks
- [ ] Monitor resources for issues

### Migration Testing Script

Use this script to test migrations:

```bash
#!/bin/bash
set -e

echo "Starting migration test..."

# Backup state
echo "Backing up state..."
cp terraform.tfstate terraform.tfstate.pre-migration

# Update provider version
echo "Updating provider version..."
terraform init -upgrade

# Run plan
echo "Running terraform plan..."
terraform plan -out=migration.tfplan

# Review plan
echo "Review the plan above. Continue? (yes/no)"
read -r response
if [ "$response" != "yes" ]; then
    echo "Migration cancelled"
    exit 1
fi

# Apply changes
echo "Applying changes..."
terraform apply migration.tfplan

# Verify
echo "Verifying resources..."
terraform state list
terraform plan

echo "Migration complete!"
```

## Rollback Procedures

If you need to rollback after an upgrade:

### Rollback Provider Version

1. **Restore previous version** in configuration:
   ```hcl
   terraform {
     required_providers {
       emma = {
         source  = "emma-community/emma"
         version = "= 0.1.0"  # Previous version
       }
     }
   }
   ```

2. **Reinitialize**:
   ```bash
   terraform init -upgrade
   ```

3. **Restore state** if needed:
   ```bash
   cp terraform.tfstate.backup terraform.tfstate
   ```

4. **Verify**:
   ```bash
   terraform plan
   ```

### Rollback State

If state was corrupted during migration:

1. **Restore from backup**:
   ```bash
   cp terraform.tfstate.backup terraform.tfstate
   ```

2. **Or restore from remote state version**:
   ```bash
   # For S3 backend
   aws s3 cp s3://bucket/path/terraform.tfstate.backup terraform.tfstate
   ```

3. **Verify state**:
   ```bash
   terraform state list
   terraform plan
   ```

## Getting Help with Migrations

If you encounter issues during migration:

1. **Check documentation**:
   - Review this migration guide
   - Check [CHANGELOG.md](../CHANGELOG.md)
   - Read release notes

2. **Search for similar issues**:
   - GitHub issues: https://github.com/emma-community/terraform-provider-emma/issues
   - Filter by version label

3. **Enable debug logging**:
   ```bash
   export TF_LOG=DEBUG
   terraform plan
   ```

4. **Create issue** with:
   - Current version
   - Target version
   - Error messages
   - Debug logs (sanitized)
   - Configuration example

5. **Contact support**:
   - Emma support for API-related issues
   - GitHub issues for provider-specific issues

## Best Practices for Upgrades

1. **Stay current**: Regularly update to latest patch versions for bug fixes

2. **Test thoroughly**: Always test upgrades in non-production first

3. **Read release notes**: Review changes before upgrading

4. **Use version constraints**: Pin to major.minor version:
   ```hcl
   version = "~> 0.1.0"  # Allows 0.1.x updates
   ```

5. **Automate testing**: Use CI/CD to test provider upgrades

6. **Monitor after upgrade**: Watch for issues after upgrading production

7. **Keep backups**: Always backup state before major upgrades

8. **Plan maintenance windows**: Schedule upgrades during low-traffic periods

9. **Document changes**: Keep internal documentation of provider versions used

10. **Communicate**: Inform team members of planned upgrades

## Future Roadmap

Planned features for future versions:

- Additional resource types as Emma API expands
- Enhanced data sources for better resource discovery
- Improved performance optimizations
- Additional validation and error handling
- Extended import capabilities
- More comprehensive testing utilities

Check the [GitHub repository](https://github.com/emma-community/terraform-provider-emma) for the latest roadmap and feature requests.

## Related Documentation

- [CHANGELOG.md](../CHANGELOG.md) - Detailed version history
- [Error Handling Guide](./ERROR_HANDLING.md) - Common errors and solutions
- [Troubleshooting Guide](./TROUBLESHOOTING.md) - Debugging and fixes
- [Architecture Guide](./ARCHITECTURE.md) - Provider design and patterns
