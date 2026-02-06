# Logging Configuration Guide

This guide explains how to configure and use logging in the Emma Terraform Provider.

## Overview

The Emma Terraform Provider uses structured logging to help you troubleshoot issues and understand what the provider is doing. Logs are written using the Terraform Plugin Framework's logging system, which integrates with Terraform's native logging.

## Log Levels

The provider uses three log levels:

### Debug Level
- **Purpose**: Detailed information for debugging
- **Content**: 
  - API requests and responses (with sensitive data redacted)
  - Retry attempts and delays
  - Async operation status checks
  - Internal state transitions

### Info Level
- **Purpose**: High-level operational information
- **Content**:
  - Resource creation, updates, and deletions
  - State transitions
  - Provider configuration

### Error Level
- **Purpose**: Error conditions and failures
- **Content**:
  - Operation failures with full context
  - API errors
  - Validation errors

## Enabling Debug Logging

### Method 1: Environment Variable (Recommended)

Set the `TF_LOG` environment variable before running Terraform:

```bash
# Enable debug logging for all Terraform operations
export TF_LOG=DEBUG
terraform apply

# Enable debug logging for provider operations only
export TF_LOG_PROVIDER=DEBUG
terraform apply
```

### Method 2: Terraform Configuration

You can also enable logging in your Terraform configuration:

```hcl
terraform {
  # This is not a standard Terraform feature, use environment variables instead
}
```

### Method 3: Command Line

Enable logging for a single command:

```bash
TF_LOG=DEBUG terraform apply
```

## Log Output Location

### Default Output
By default, logs are written to stderr and appear in your terminal.

### File Output
To write logs to a file, set the `TF_LOG_PATH` environment variable:

```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=terraform.log
terraform apply
```

## Log Format

Logs are structured in JSON format for easy parsing:

```json
{
  "@level": "debug",
  "@message": "API Request",
  "@module": "provider.terraform-provider-emma",
  "@timestamp": "2024-01-15T10:30:45.123456Z",
  "method": "POST",
  "path": "/volumes",
  "body": {
    "name": "my-volume",
    "size_gb": 100,
    "password": "[REDACTED]"
  }
}
```

## Sensitive Data Sanitization

The provider automatically redacts sensitive data from logs to protect your credentials and secrets.

### Redacted Fields

The following field names are automatically redacted:
- `password`
- `token`
- `secret`
- `key`
- `access_token`
- `refresh_token`
- `api_key`
- `private_key`
- `client_secret`
- `authorization`
- `bearer`
- `credentials`

Field name matching is case-insensitive and includes partial matches (e.g., `user_password` is redacted).

### Redacted Patterns

The following patterns in string values are automatically redacted:
- Bearer tokens: `Bearer abc123` → `Bearer [REDACTED]`
- Basic authentication: `Basic dXNlcjpwYXNz` → `Basic [REDACTED]`

### Example

**Before sanitization:**
```json
{
  "username": "admin",
  "password": "secret123",
  "api_key": "key_abc123"
}
```

**After sanitization:**
```json
{
  "username": "admin",
  "password": "[REDACTED]",
  "api_key": "[REDACTED]"
}
```

## Common Logging Scenarios

### Debugging API Errors

Enable debug logging to see the full API request and response:

```bash
TF_LOG=DEBUG terraform apply 2>&1 | grep "API"
```

### Tracking Resource State Changes

Enable info logging to see resource lifecycle events:

```bash
TF_LOG=INFO terraform apply 2>&1 | grep "Resource"
```

### Investigating Retry Behavior

Enable debug logging to see retry attempts:

```bash
TF_LOG=DEBUG terraform apply 2>&1 | grep "Retry"
```

### Monitoring Async Operations

Enable debug logging to see async operation status:

```bash
TF_LOG=DEBUG terraform apply 2>&1 | grep "Async"
```

## Filtering Logs

### Using grep

Filter logs by level:
```bash
TF_LOG=DEBUG terraform apply 2>&1 | grep "@level.*debug"
TF_LOG=DEBUG terraform apply 2>&1 | grep "@level.*info"
TF_LOG=DEBUG terraform apply 2>&1 | grep "@level.*error"
```

Filter logs by resource type:
```bash
TF_LOG=DEBUG terraform apply 2>&1 | grep "emma_volume"
TF_LOG=DEBUG terraform apply 2>&1 | grep "emma_vm"
```

### Using jq

Parse and filter JSON logs:
```bash
# Extract all error messages
TF_LOG=DEBUG terraform apply 2>&1 | jq 'select(."@level" == "error")'

# Extract API requests
TF_LOG=DEBUG terraform apply 2>&1 | jq 'select(."@message" == "API Request")'

# Extract resource operations
TF_LOG=DEBUG terraform apply 2>&1 | jq 'select(.resource_type != null)'
```

## Performance Considerations

### Debug Logging Overhead

Debug logging adds minimal overhead to provider operations:
- API request/response logging: ~1-5ms per request
- State transition logging: <1ms per transition
- Sanitization: <1ms per log entry

### Production Recommendations

For production use:
1. Use `INFO` level logging by default
2. Enable `DEBUG` logging only when troubleshooting
3. Rotate log files to prevent disk space issues
4. Consider using log aggregation tools for centralized logging

## Troubleshooting

### No Logs Appearing

1. Check that `TF_LOG` is set:
   ```bash
   echo $TF_LOG
   ```

2. Verify Terraform version (requires Terraform >= 1.0):
   ```bash
   terraform version
   ```

3. Check that the provider is being used:
   ```bash
   terraform providers
   ```

### Logs Too Verbose

1. Reduce log level:
   ```bash
   export TF_LOG=INFO  # Instead of DEBUG
   ```

2. Filter logs using grep or jq (see above)

3. Use `TF_LOG_PROVIDER` instead of `TF_LOG` to only log provider operations

### Sensitive Data in Logs

If you find sensitive data in logs that should be redacted:

1. Check that you're using the latest provider version
2. Report the issue with an example (redact the actual sensitive data)
3. As a workaround, avoid logging the specific operation

## Best Practices

1. **Always enable logging when troubleshooting**: Debug logs provide valuable context for diagnosing issues

2. **Use structured logging**: The JSON format makes it easy to parse and analyze logs programmatically

3. **Rotate log files**: If using `TF_LOG_PATH`, implement log rotation to prevent disk space issues

4. **Sanitize before sharing**: Even though the provider redacts sensitive data, always review logs before sharing them

5. **Use appropriate log levels**: 
   - Development: `DEBUG`
   - Production: `INFO`
   - Troubleshooting: `DEBUG`

6. **Combine with Terraform logs**: Provider logs work best when combined with Terraform's own logs

## Examples

### Example 1: Debug a Failed Resource Creation

```bash
# Enable debug logging
export TF_LOG=DEBUG
export TF_LOG_PATH=debug.log

# Run terraform
terraform apply

# Review the logs
cat debug.log | jq 'select(.resource_type == "emma_volume")'
```

### Example 2: Monitor Async Operations

```bash
# Enable debug logging and filter for async operations
TF_LOG=DEBUG terraform apply 2>&1 | grep "Async operation status"
```

### Example 3: Track API Rate Limiting

```bash
# Enable debug logging and filter for retry attempts
TF_LOG=DEBUG terraform apply 2>&1 | grep "Retry attempt"
```

## Additional Resources

- [Terraform Debugging Documentation](https://www.terraform.io/docs/internals/debugging.html)
- [Terraform Plugin Framework Logging](https://developer.hashicorp.com/terraform/plugin/framework/logging)
- [Emma Provider Documentation](./index.md)

## Support

If you encounter issues with logging:

1. Check this documentation
2. Review the [troubleshooting guide](./TROUBLESHOOTING.md)
3. Open an issue on [GitHub](https://github.com/emma-community/terraform-provider-emma/issues)
