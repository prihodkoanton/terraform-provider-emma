# Emma Terraform Provider - Architecture Guide

## Overview

This guide provides an architectural overview of the Emma Terraform Provider, focusing on the shared utilities and patterns introduced to improve code quality, maintainability, and consistency across all resources.

The provider is built using the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) and follows HashiCorp's best practices for provider development.

## Architecture Principles

1. **Separation of Concerns**: Business logic, API interaction, and Terraform-specific code are clearly separated
2. **Code Reusability**: Common patterns are extracted into shared utilities
3. **Consistency**: All resources follow the same patterns for error handling, state management, and validation
4. **Testability**: Property-based testing validates universal correctness properties
5. **Maintainability**: Clear naming conventions and documentation make the codebase easy to navigate

## Directory Structure

```
terraform-provider-emma/
├── internal/emma/
│   ├── common/                      # Shared utilities (NEW)
│   │   ├── errors/                  # Centralized error handling
│   │   ├── convert/                 # Type conversion utilities
│   │   ├── state/                   # State management helpers
│   │   ├── async/                   # Async operation handling
│   │   ├── retry/                   # Retry logic with backoff
│   │   ├── logging/                 # Structured logging utilities
│   │   └── testing/                 # Test utilities and fixtures
│   ├── validation/                  # Custom validators
│   ├── *_resource.go                # Resource implementations
│   ├── *_data_source.go             # Data source implementations
│   └── provider.go                  # Provider configuration
├── docs/                            # Documentation
├── examples/                        # Terraform examples
└── main.go                          # Entry point
```

## Core Components

### 1. Error Handling (`internal/emma/common/errors/`)

Centralized error handling provides consistent, informative error messages across all resources.

#### When to Use

- **Always** use `ErrorBuilder` when creating errors in resource operations
- Use `MapHTTPError()` to convert API errors to user-friendly messages
- Use `IsRetryable()` to determine if an error should be retried

#### Example Usage

```go
import "github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"

// Creating a resource error
err := errors.NewError("emma_volume", "Create").
    WithID(volumeID).
    WithStatusCode(resp.StatusCode).
    WithAPIError(apiErr).
    WithMessage("Failed to create volume").
    Build()

// Mapping HTTP errors
userMessage := errors.MapHTTPError(statusCode, apiErrorMessage)

// Checking if error is retryable
if errors.IsRetryable(statusCode) {
    // Retry the operation
}
```

#### Key Types

- `ResourceError`: Structured error with context (resource type, ID, operation, status code)
- `ErrorBuilder`: Fluent API for building errors
- `MapHTTPError()`: Maps HTTP status codes to user-friendly messages
- `IsRetryable()`: Determines if an error should be retried

### 2. Type Conversion (`internal/emma/common/convert/`)

Shared utilities for converting between SDK types and Terraform types.

#### When to Use

- Converting between Emma SDK types (e.g., `*int32`) and Terraform types (e.g., `types.Int64`)
- Handling null and optional values consistently
- Converting nested objects and lists

#### Example Usage

```go
import "github.com/emma-community/terraform-provider-emma/internal/emma/common/convert"

// Basic conversions
tfString := convert.Int32ToString(sdkInt32Ptr)
sdkInt32, err := convert.StringToInt32(tfString)
tfInt64 := convert.Int32ToInt64(sdkInt32Ptr)

// Null-safe conversions
tfString := convert.StringPointerToString(sdkStringPtr)
tfBool := convert.BoolPointerToBool(sdkBoolPtr)

// Nested object conversions
converter := convert.NewObjectConverter(ctx)
tfObject, err := converter.ToObject(attrTypes, sdkStruct)
tfList, err := converter.ToList(elementType, sdkSlice)
```

#### Key Functions

- `Int32ToString()`, `StringToInt32()`: Convert between int32 and string
- `Int64ToInt32()`, `Int32ToInt64()`: Convert between int sizes with bounds checking
- `StringPointerToString()`, `BoolPointerToBool()`: Null-safe pointer conversions
- `ObjectConverter`: Handles nested object and list conversions

### 3. State Management (`internal/emma/common/state/`)

Helpers for consistent state management across all resources.

#### When to Use

- Handling 404 responses (resource deleted outside Terraform)
- Updating computed attributes from API responses
- Preserving user-specified values during updates
- Detecting drift between state and actual infrastructure

#### Example Usage

```go
import "github.com/emma-community/terraform-provider-emma/internal/emma/common/state"

// Handle 404 - remove from state
if statusCode == 404 {
    sm := state.NewStateManager(ctx)
    sm.RemoveFromState(&resp.State)
    return
}

// Update only computed attributes
sm := state.NewStateManager(ctx)
computedFields := []string{"status", "created_at", "updated_at"}
err := sm.UpdateComputedAttributes(&currentState, apiResponse, computedFields)

// Preserve user values during update
userFields := []string{"name", "description", "tags"}
err := sm.PreserveUserValues(&currentState, &newState, userFields)

// Detect drift
detector := state.NewDriftDetector()
drifts, err := detector.DetectDrift(stateValue, apiValue)
if len(drifts) > 0 {
    tflog.Warn(ctx, "Drift detected", map[string]interface{}{
        "fields": drifts,
    })
}
```

#### Key Types

- `StateManager`: Provides state operation helpers
- `DriftDetector`: Detects differences between state and API

### 4. Async Operations (`internal/emma/common/async/`)

Unified polling mechanism for long-running operations.

#### When to Use

- Operations that return immediately but complete asynchronously
- Hardware modifications (CPU, RAM changes)
- Volume resizing
- Security group synchronization
- Any operation with a status that needs polling

#### Example Usage

```go
import (
    "github.com/emma-community/terraform-provider-emma/internal/emma/common/async"
    "time"
)

// Configure poller
config := async.PollerConfig{
    Timeout:      30 * time.Minute,
    PollInterval: 10 * time.Second,
    StatusChecker: func(ctx context.Context) (string, error) {
        resp, _, err := client.VolumesAPI.GetVolume(auth, volumeID).Execute()
        if err != nil {
            return "", err
        }
        return *resp.Status, nil
    },
    TargetStates:  []string{"available", "in-use"},
    FailureStates: []string{"error", "failed"},
}

// Poll until complete
poller := async.NewPoller(config)
if err := poller.Poll(ctx); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

#### Key Types

- `PollerConfig`: Configuration for async polling
- `Poller`: Handles polling with timeout and failure detection

### 5. Retry Logic (`internal/emma/common/retry/`)

Exponential backoff retry mechanism for transient failures.

#### When to Use

- API calls that may fail due to rate limiting (429)
- Transient network errors
- Server errors (5xx)
- Service unavailable (503)

#### Example Usage

```go
import (
    "github.com/emma-community/terraform-provider-emma/internal/emma/common/retry"
    "github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"
    "time"
)

// Configure retry behavior
config := retry.DefaultRetryConfig()
config.MaxAttempts = 5
config.InitialDelay = 2 * time.Second
config.MaxDelay = 60 * time.Second
config.ShouldRetry = func(err error) bool {
    if resErr, ok := err.(*errors.ResourceError); ok {
        return errors.IsRetryable(resErr.StatusCode)
    }
    return false
}

// Retry operation
err := retry.Retry(ctx, config, func() error {
    _, _, err := client.VolumesAPI.CreateVolume(auth).
        VolumeCreate(volumeRequest).
        Execute()
    return err
})
```

#### Key Types

- `RetryConfig`: Configuration for retry behavior
- `DefaultRetryConfig()`: Sensible defaults for most use cases
- `Retry()`: Executes operation with exponential backoff

### 6. Logging (`internal/emma/common/logging/`)

Structured logging with sensitive data sanitization.

#### When to Use

- Logging API requests and responses (debug level)
- Logging state transitions (info level)
- Logging errors (error level)
- Any logging that might contain sensitive data

#### Example Usage

```go
import "github.com/emma-community/terraform-provider-emma/internal/emma/common/logging"

// Create logger
logger := logging.NewLogger(ctx)

// Log API request (automatically sanitizes)
logger.LogAPIRequest("POST", "/volumes", requestBody)

// Log API response
logger.LogAPIResponse("POST", "/volumes", statusCode, responseBody)

// Log state transition
logger.LogStateTransition("emma_volume", volumeID, "creating", "available")

// Log error with context
logger.LogError("Failed to create volume", map[string]interface{}{
    "volume_id": volumeID,
    "error":     err.Error(),
})

// Sanitize sensitive data manually
sanitized := logging.SanitizeValue(data)
```

#### Key Functions

- `NewLogger()`: Creates a logger instance
- `LogAPIRequest()`, `LogAPIResponse()`: Log API interactions with sanitization
- `LogStateTransition()`: Log resource state changes
- `LogError()`: Log errors with context
- `SanitizeValue()`: Remove sensitive data from any value

### 7. Validation (`internal/emma/validation/`)

Enhanced validation framework with cross-field validators.

#### When to Use

- Validating mutually exclusive fields
- Validating that at least one of several fields is set
- Custom format validation (Emma-specific patterns)
- Range validation with custom messages

#### Example Usage

```go
import "github.com/emma-community/terraform-provider-emma/internal/emma/validation"

// In resource schema
"volume_type": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        validation.MutuallyExclusive([]string{"volume_type", "volume_config_id"}),
        validation.OneOf([]string{"ssd", "ssd-plus", "hdd"}),
    },
}

"instance_id": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        validation.RequiresOneOf([]string{"instance_id", "spot_instance_id"}),
    },
}
```

#### Key Validators

- `MutuallyExclusive`: Ensures only one field in a group is set
- `RequiresOneOf`: Ensures at least one field in a group is set
- Custom validators for Emma-specific formats

### 8. Testing Utilities (`internal/emma/common/testing/`)

Shared test fixtures and helpers for consistent testing.

#### When to Use

- Writing unit tests for resources
- Writing property-based tests
- Writing integration tests
- Generating test data

#### Example Usage

```go
import (
    "github.com/emma-community/terraform-provider-emma/internal/emma/common/testing/fixtures"
    "github.com/emma-community/terraform-provider-emma/internal/emma/common/testing"
)

// Use fixtures for consistent test data
func TestVolumeCreate(t *testing.T) {
    volume := fixtures.VolumeFixture()
    // Use volume in test
}

// Use generators for property tests
func TestVolumeValidation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("valid volumes pass validation", prop.ForAll(
        func(config map[string]interface{}) bool {
            // Test validation
            return true
        },
        fixtures.VolumeConfigGen(),
    ))
    
    properties.TestingRun(t)
}

// Use helpers for common test operations
func TestVolumeConversion(t *testing.T) {
    testing.AssertNoError(t, err)
    testing.AssertEqual(t, expected, actual)
}
```

#### Key Components

- `fixtures.VolumeFixture()`, `fixtures.VmFixture()`: Consistent test data
- `fixtures.VolumeConfigGen()`: Property test generators
- `testing.AssertNoError()`, `testing.AssertEqual()`: Test helpers

## Migration Guide

### Migrating Existing Resources

Follow these steps to migrate an existing resource to use the new utilities:

#### Step 1: Update Error Handling

**Before:**
```go
if err != nil {
    resp.Diagnostics.AddError(
        "Error creating volume",
        fmt.Sprintf("Could not create volume: %s", err.Error()),
    )
    return
}
```

**After:**
```go
if err != nil {
    resErr := errors.NewError("emma_volume", "Create").
        WithAPIError(err.Error()).
        WithMessage("Failed to create volume").
        Build()
    resp.Diagnostics.AddError(
        "Error creating volume",
        resErr.Error(),
    )
    return
}
```

#### Step 2: Update Type Conversions

**Before:**
```go
var volumeGb int32
if !plan.VolumeGb.IsNull() {
    volumeGb = int32(plan.VolumeGb.ValueInt64())
}
```

**After:**
```go
volumeGb, err := convert.Int64ToInt32(plan.VolumeGb)
if err != nil {
    resp.Diagnostics.AddError("Invalid volume size", err.Error())
    return
}
```

#### Step 3: Update State Management

**Before:**
```go
if resp.StatusCode == 404 {
    resp.State.RemoveResource(ctx)
    return
}
```

**After:**
```go
if resp.StatusCode == 404 {
    sm := state.NewStateManager(ctx)
    sm.RemoveFromState(&resp.State)
    return
}
```

#### Step 4: Add Async Polling

**Before:**
```go
// Manual polling loop
for i := 0; i < 60; i++ {
    time.Sleep(10 * time.Second)
    status, err := checkStatus()
    if status == "available" {
        break
    }
}
```

**After:**
```go
config := async.PollerConfig{
    Timeout:      30 * time.Minute,
    PollInterval: 10 * time.Second,
    StatusChecker: func(ctx context.Context) (string, error) {
        return checkStatus()
    },
    TargetStates:  []string{"available"},
    FailureStates: []string{"error"},
}
poller := async.NewPoller(config)
if err := poller.Poll(ctx); err != nil {
    return err
}
```

#### Step 5: Add Retry Logic

**Before:**
```go
resp, _, err := client.API.CreateResource(auth).Execute()
if err != nil {
    return err
}
```

**After:**
```go
var resp *Resource
err := retry.Retry(ctx, retry.DefaultRetryConfig(), func() error {
    var apiErr error
    resp, _, apiErr = client.API.CreateResource(auth).Execute()
    return apiErr
})
if err != nil {
    return err
}
```

#### Step 6: Add Structured Logging

**Before:**
```go
tflog.Debug(ctx, "Creating volume", map[string]interface{}{
    "name": volumeName,
})
```

**After:**
```go
logger := logging.NewLogger(ctx)
logger.LogAPIRequest("POST", "/volumes", requestBody)
logger.LogStateTransition("emma_volume", volumeID, "creating", "available")
```

### Migration Checklist

- [ ] Replace manual error handling with `ErrorBuilder`
- [ ] Replace manual type conversions with `convert` package
- [ ] Replace manual state operations with `StateManager`
- [ ] Replace manual polling loops with `Poller`
- [ ] Add retry logic for API calls
- [ ] Add structured logging with sanitization
- [ ] Update tests to use fixtures and generators
- [ ] Add property-based tests for universal properties

## Best Practices

### Error Handling

1. **Always include context**: Use `ErrorBuilder` to include resource type, operation, and ID
2. **Map HTTP errors**: Use `MapHTTPError()` for user-friendly messages
3. **Preserve original errors**: Use `WithCause()` to preserve the error chain
4. **Check retryability**: Use `IsRetryable()` before implementing retry logic

### Type Conversion

1. **Use shared utilities**: Never write custom conversion functions
2. **Handle nulls consistently**: Use null-safe conversion functions
3. **Validate bounds**: Use functions with bounds checking for numeric conversions
4. **Check errors**: Always check conversion errors

### State Management

1. **Handle 404s consistently**: Always use `StateManager.RemoveFromState()`
2. **Preserve user values**: Use `PreserveUserValues()` during updates
3. **Update computed fields**: Use `UpdateComputedAttributes()` for API-computed values
4. **Detect drift**: Use `DriftDetector` to warn about external changes

### Async Operations

1. **Configure timeouts**: Always set reasonable timeouts
2. **Define failure states**: Specify what states indicate failure
3. **Use appropriate intervals**: Balance responsiveness with API load
4. **Handle cancellation**: Respect context cancellation

### Retry Logic

1. **Use defaults**: Start with `DefaultRetryConfig()`
2. **Check retryability**: Only retry transient errors
3. **Set max attempts**: Prevent infinite retry loops
4. **Log retry attempts**: Help with debugging

### Logging

1. **Use appropriate levels**: Debug for API calls, Info for state changes, Error for failures
2. **Sanitize sensitive data**: Always use logging utilities for sensitive data
3. **Include context**: Add resource type, ID, and operation to logs
4. **Structure logs**: Use structured logging for machine parsing

### Testing

1. **Use fixtures**: Consistent test data across all tests
2. **Write property tests**: Validate universal properties
3. **Test error paths**: Ensure error handling works correctly
4. **Use generators**: Generate diverse test inputs

## Common Patterns

### Resource CRUD Pattern

```go
func (r *VolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan VolumeResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    logger := logging.NewLogger(ctx)
    
    // Convert plan to API request
    volumeRequest := convertPlanToRequest(plan)
    
    // Retry API call
    var volume *emmaSdk.Volume
    err := retry.Retry(ctx, retry.DefaultRetryConfig(), func() error {
        logger.LogAPIRequest("POST", "/volumes", volumeRequest)
        var apiErr error
        volume, _, apiErr = r.client.VolumesAPI.CreateVolume(auth).
            VolumeCreate(volumeRequest).
            Execute()
        return apiErr
    })
    
    if err != nil {
        resErr := errors.NewError("emma_volume", "Create").
            WithAPIError(err.Error()).
            Build()
        resp.Diagnostics.AddError("Error creating volume", resErr.Error())
        return
    }
    
    logger.LogAPIResponse("POST", "/volumes", 201, volume)
    
    // Poll for completion
    poller := async.NewPoller(async.PollerConfig{
        Timeout:      30 * time.Minute,
        PollInterval: 10 * time.Second,
        StatusChecker: func(ctx context.Context) (string, error) {
            v, _, err := r.client.VolumesAPI.GetVolume(auth, *volume.Id).Execute()
            if err != nil {
                return "", err
            }
            return *v.Status, nil
        },
        TargetStates:  []string{"available"},
        FailureStates: []string{"error"},
    })
    
    if err := poller.Poll(ctx); err != nil {
        resp.Diagnostics.AddError("Error waiting for volume", err.Error())
        return
    }
    
    logger.LogStateTransition("emma_volume", fmt.Sprintf("%d", *volume.Id), "creating", "available")
    
    // Convert response to state
    state := convertResponseToState(volume)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### Data Source Read Pattern

```go
func (d *VolumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var config VolumeDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }

    logger := logging.NewLogger(ctx)
    volumeID, _ := convert.StringToInt32(config.ID)
    
    // Retry API call
    var volume *emmaSdk.Volume
    err := retry.Retry(ctx, retry.DefaultRetryConfig(), func() error {
        logger.LogAPIRequest("GET", fmt.Sprintf("/volumes/%d", volumeID), nil)
        var apiErr error
        volume, _, apiErr = d.client.VolumesAPI.GetVolume(auth, volumeID).Execute()
        return apiErr
    })
    
    if err != nil {
        resErr := errors.NewError("emma_volume", "Read").
            WithID(config.ID.ValueString()).
            WithAPIError(err.Error()).
            Build()
        resp.Diagnostics.AddError("Error reading volume", resErr.Error())
        return
    }
    
    logger.LogAPIResponse("GET", fmt.Sprintf("/volumes/%d", volumeID), 200, volume)
    
    // Convert response to state
    state := convertResponseToState(volume)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

## Performance Considerations

1. **Minimize API calls**: Use state efficiently to avoid unnecessary reads
2. **Batch operations**: Where API supports it, batch multiple operations
3. **Cache data sources**: Results are cached within a single Terraform run
4. **Optimize polling**: Use appropriate intervals to balance responsiveness and load
5. **Parallel operations**: Terraform handles parallelism; ensure resources are safe

## Security Considerations

1. **Mark sensitive attributes**: Use `Sensitive: true` in schema
2. **Sanitize logs**: Always use logging utilities for sensitive data
3. **Validate certificates**: SSL/TLS validation is enabled by default
4. **Clear sensitive data**: Remove from memory after use
5. **Document security**: Include security considerations in resource docs

## Troubleshooting

### Common Issues

**Issue**: Errors are not descriptive enough
- **Solution**: Use `ErrorBuilder` with full context (resource type, ID, operation)

**Issue**: Type conversion panics on null values
- **Solution**: Use null-safe conversion functions from `convert` package

**Issue**: Resources not removed from state on 404
- **Solution**: Use `StateManager.RemoveFromState()` in Read operations

**Issue**: Async operations timeout
- **Solution**: Increase timeout in `PollerConfig` or check API status

**Issue**: Too many API calls (rate limiting)
- **Solution**: Implement retry logic with exponential backoff

**Issue**: Sensitive data in logs
- **Solution**: Use logging utilities that automatically sanitize

## Additional Resources

- [Terraform Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Emma API Documentation](https://docs.emma.ms/)
- [API Reference](./API_REFERENCE.md)
- [Testing Guide](./TESTING_GUIDE.md)
- [Logging Configuration](./LOGGING.md)

## Contributing

When adding new resources or modifying existing ones:

1. Follow the patterns described in this guide
2. Use shared utilities instead of duplicating code
3. Add property-based tests for universal properties
4. Update documentation with examples
5. Run full test suite before submitting
6. Follow Go best practices and idioms

## Questions and Support

For questions about the architecture or utilities:
- Review the API reference documentation
- Check existing resource implementations for examples
- Consult the testing guide for test patterns
- Open an issue for clarification or improvements
