# Emma Terraform Provider - API Reference

## Overview

This document provides a comprehensive reference for all exported functions, types, and utilities in the Emma Terraform Provider's shared libraries.

## Table of Contents

- [Error Handling](#error-handling)
- [Type Conversion](#type-conversion)
- [State Management](#state-management)
- [Async Operations](#async-operations)
- [Retry Logic](#retry-logic)
- [Logging](#logging)
- [Validation](#validation)
- [Testing Utilities](#testing-utilities)

---

## Error Handling

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/errors`

### Types

#### ResourceError

Represents a provider error with full context.

```go
type ResourceError struct {
    ResourceType string  // e.g., "emma_volume"
    ResourceID   string  // e.g., "12345"
    Operation    string  // e.g., "Create", "Update", "Delete"
    StatusCode   int     // HTTP status code
    APIError     string  // Original API error message
    Message      string  // User-friendly message
    Cause        error   // Underlying error
}
```

**Methods:**

- `Error() string`: Returns formatted error message

**Example:**
```go
err := &ResourceError{
    ResourceType: "emma_volume",
    Operation:    "Create",
    Message:      "Failed to create volume",
}
fmt.Println(err.Error()) // "Create emma_volume failed: Failed to create volume"
```

#### ErrorBuilder

Fluent API for building ResourceError instances.

```go
type ErrorBuilder struct {
    err *ResourceError
}
```

### Functions

#### NewError

Creates a new ErrorBuilder.

```go
func NewError(resourceType, operation string) *ErrorBuilder
```

**Parameters:**
- `resourceType`: The Terraform resource type (e.g., "emma_volume")
- `operation`: The operation being performed (e.g., "Create", "Read", "Update", "Delete")

**Returns:** `*ErrorBuilder`


**Example:**
```go
builder := errors.NewError("emma_volume", "Create")
```

#### ErrorBuilder Methods

##### WithID

Adds resource ID to the error.

```go
func (b *ErrorBuilder) WithID(id string) *ErrorBuilder
```

**Example:**
```go
builder.WithID("12345")
```

##### WithStatusCode

Adds HTTP status code to the error.

```go
func (b *ErrorBuilder) WithStatusCode(code int) *ErrorBuilder
```

**Example:**
```go
builder.WithStatusCode(404)
```

##### WithAPIError

Adds the original API error message.

```go
func (b *ErrorBuilder) WithAPIError(apiErr string) *ErrorBuilder
```

**Example:**
```go
builder.WithAPIError("Volume not found")
```

##### WithMessage

Adds a user-friendly error message.

```go
func (b *ErrorBuilder) WithMessage(msg string) *ErrorBuilder
```

**Example:**
```go
builder.WithMessage("Failed to create volume")
```

##### WithCause

Adds the underlying error cause.

```go
func (b *ErrorBuilder) WithCause(err error) *ErrorBuilder
```

**Example:**
```go
builder.WithCause(originalErr)
```

##### Build

Builds and returns the ResourceError.

```go
func (b *ErrorBuilder) Build() *ResourceError
```

**Example:**
```go
err := errors.NewError("emma_volume", "Create").
    WithID("12345").
    WithStatusCode(500).
    WithAPIError("Internal server error").
    WithMessage("Failed to create volume").
    Build()
```

#### MapHTTPError

Maps HTTP status codes to user-friendly error messages.

```go
func MapHTTPError(statusCode int, apiMessage string) string
```

**Parameters:**
- `statusCode`: HTTP status code from API response
- `apiMessage`: Original error message from API

**Returns:** User-friendly error message

**Supported Status Codes:**
- `400`: Bad Request
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Not Found
- `409`: Conflict
- `422`: Unprocessable Entity
- `429`: Too Many Requests
- `500`: Internal Server Error
- `503`: Service Unavailable

**Example:**
```go
message := errors.MapHTTPError(404, "Volume not found")
// Returns: "Resource not found. It may have been deleted."
```

#### IsRetryable

Determines if an HTTP error should be retried.

```go
func IsRetryable(statusCode int) bool
```

**Parameters:**
- `statusCode`: HTTP status code from API response

**Returns:** `true` if the error is retryable, `false` otherwise

**Retryable Status Codes:**
- `429`: Too Many Requests
- `503`: Service Unavailable
- `5xx`: Server errors (500-599)

**Example:**
```go
if errors.IsRetryable(statusCode) {
    // Implement retry logic
}
```

---

## Type Conversion

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/convert`

### Basic Type Conversions

#### Int32ToString

Converts SDK int32 pointer to Terraform string.

```go
func Int32ToString(value *int32) types.String
```

**Parameters:**
- `value`: Pointer to int32 value (can be nil)

**Returns:** `types.String` (null if input is nil)

**Example:**
```go
id := int32(12345)
tfString := convert.Int32ToString(&id)
// tfString.ValueString() == "12345"

tfNull := convert.Int32ToString(nil)
// tfNull.IsNull() == true
```

#### StringToInt32

Converts Terraform string to int32 with validation.

```go
func StringToInt32(value types.String) (int32, error)
```

**Parameters:**
- `value`: Terraform string value

**Returns:** 
- `int32`: Converted value
- `error`: Error if value is null, unknown, or invalid format

**Example:**
```go
tfString := types.StringValue("12345")
id, err := convert.StringToInt32(tfString)
if err != nil {
    // Handle error
}
// id == 12345
```

#### Int64ToInt32

Converts Terraform int64 to int32 with bounds checking.

```go
func Int64ToInt32(value types.Int64) (int32, error)
```

**Parameters:**
- `value`: Terraform int64 value

**Returns:**
- `int32`: Converted value
- `error`: Error if value is null, unknown, or out of int32 range

**Example:**
```go
tfInt64 := types.Int64Value(100)
id, err := convert.Int64ToInt32(tfInt64)
if err != nil {
    // Handle error
}
// id == 100
```

#### Int32ToInt64

Converts SDK int32 pointer to Terraform int64.

```go
func Int32ToInt64(value *int32) types.Int64
```

**Parameters:**
- `value`: Pointer to int32 value (can be nil)

**Returns:** `types.Int64` (null if input is nil)

**Example:**
```go
size := int32(100)
tfInt64 := convert.Int32ToInt64(&size)
// tfInt64.ValueInt64() == 100
```

#### StringPointerToString

Converts SDK string pointer to Terraform string.

```go
func StringPointerToString(value *string) types.String
```

**Parameters:**
- `value`: Pointer to string value (can be nil)

**Returns:** `types.String` (null if input is nil)

**Example:**
```go
name := "test-volume"
tfString := convert.StringPointerToString(&name)
// tfString.ValueString() == "test-volume"
```

#### BoolPointerToBool

Converts SDK bool pointer to Terraform bool.

```go
func BoolPointerToBool(value *bool) types.Bool
```

**Parameters:**
- `value`: Pointer to bool value (can be nil)

**Returns:** `types.Bool` (null if input is nil)

**Example:**
```go
isSystem := false
tfBool := convert.BoolPointerToBool(&isSystem)
// tfBool.ValueBool() == false
```

### Nested Object Conversions

#### ObjectConverter

Provides utilities for converting nested objects and lists.

```go
type ObjectConverter struct {
    ctx context.Context
}
```

#### NewObjectConverter

Creates a new ObjectConverter.

```go
func NewObjectConverter(ctx context.Context) *ObjectConverter
```

**Parameters:**
- `ctx`: Context for the conversion

**Returns:** `*ObjectConverter`

**Example:**
```go
converter := convert.NewObjectConverter(ctx)
```

#### ToObject

Converts a struct to Terraform object.

```go
func (c *ObjectConverter) ToObject(attrTypes map[string]attr.Type, value interface{}) (types.Object, error)
```

**Parameters:**
- `attrTypes`: Map of attribute names to types
- `value`: Struct to convert

**Returns:**
- `types.Object`: Converted object
- `error`: Error if conversion fails

**Example:**
```go
attrTypes := map[string]attr.Type{
    "name": types.StringType,
    "size": types.Int64Type,
}
tfObject, err := converter.ToObject(attrTypes, sdkStruct)
```

#### ToList

Converts a slice to Terraform list.

```go
func (c *ObjectConverter) ToList(elementType attr.Type, values interface{}) (types.List, error)
```

**Parameters:**
- `elementType`: Type of list elements
- `values`: Slice to convert

**Returns:**
- `types.List`: Converted list
- `error`: Error if conversion fails

**Example:**
```go
tfList, err := converter.ToList(types.StringType, []string{"a", "b", "c"})
```

---

## State Management

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/state`

### StateManager

Provides utilities for common state operations.

```go
type StateManager struct {
    ctx context.Context
}
```

#### NewStateManager

Creates a new StateManager.

```go
func NewStateManager(ctx context.Context) *StateManager
```

**Parameters:**
- `ctx`: Context for state operations

**Returns:** `*StateManager`

**Example:**
```go
sm := state.NewStateManager(ctx)
```

#### RemoveFromState

Removes a resource from Terraform state (typically for 404 responses).

```go
func (sm *StateManager) RemoveFromState(state *tfsdk.State)
```

**Parameters:**
- `state`: Pointer to the Terraform state

**Example:**
```go
if statusCode == 404 {
    sm := state.NewStateManager(ctx)
    sm.RemoveFromState(&resp.State)
    return
}
```

#### UpdateComputedAttributes

Updates only computed attributes from API response.

```go
func (sm *StateManager) UpdateComputedAttributes(state interface{}, apiResponse interface{}, computedFields []string) error
```

**Parameters:**
- `state`: Current state struct
- `apiResponse`: API response struct
- `computedFields`: List of field names to update

**Returns:** `error` if update fails

**Example:**
```go
computedFields := []string{"status", "created_at", "updated_at"}
err := sm.UpdateComputedAttributes(&currentState, apiResponse, computedFields)
```

#### PreserveUserValues

Preserves user-specified values during state updates.

```go
func (sm *StateManager) PreserveUserValues(currentState, newState interface{}, userFields []string) error
```

**Parameters:**
- `currentState`: Current state struct
- `newState`: New state struct
- `userFields`: List of field names to preserve

**Returns:** `error` if preservation fails

**Example:**
```go
userFields := []string{"name", "description", "tags"}
err := sm.PreserveUserValues(&currentState, &newState, userFields)
```

### DriftDetector

Detects differences between state and actual infrastructure.

```go
type DriftDetector struct{}
```

#### NewDriftDetector

Creates a new DriftDetector.

```go
func NewDriftDetector() *DriftDetector
```

**Returns:** `*DriftDetector`

**Example:**
```go
detector := state.NewDriftDetector()
```

#### DetectDrift

Compares state with API response and returns differences.

```go
func (dd *DriftDetector) DetectDrift(stateValue, apiValue interface{}) ([]string, error)
```

**Parameters:**
- `stateValue`: Value from Terraform state
- `apiValue`: Value from API response

**Returns:**
- `[]string`: List of field names that have drifted
- `error`: Error if comparison fails

**Example:**
```go
drifts, err := detector.DetectDrift(stateValue, apiValue)
if len(drifts) > 0 {
    tflog.Warn(ctx, "Drift detected", map[string]interface{}{
        "fields": drifts,
    })
}
```

---

## Async Operations

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/async`

### PollerConfig

Configuration for async operation polling.

```go
type PollerConfig struct {
    Timeout       time.Duration
    PollInterval  time.Duration
    StatusChecker func(ctx context.Context) (string, error)
    TargetStates  []string
    FailureStates []string
}
```

**Fields:**
- `Timeout`: Maximum time to wait for operation completion
- `PollInterval`: Time between status checks
- `StatusChecker`: Function that returns current status
- `TargetStates`: States that indicate successful completion
- `FailureStates`: States that indicate failure

**Example:**
```go
config := async.PollerConfig{
    Timeout:      30 * time.Minute,
    PollInterval: 10 * time.Second,
    StatusChecker: func(ctx context.Context) (string, error) {
        resp, _, err := client.GetVolume(auth, volumeID).Execute()
        if err != nil {
            return "", err
        }
        return *resp.Status, nil
    },
    TargetStates:  []string{"available", "in-use"},
    FailureStates: []string{"error", "failed"},
}
```

### Poller

Handles async operation polling.

```go
type Poller struct {
    config PollerConfig
}
```

#### NewPoller

Creates a new Poller.

```go
func NewPoller(config PollerConfig) *Poller
```

**Parameters:**
- `config`: Poller configuration

**Returns:** `*Poller`

**Example:**
```go
poller := async.NewPoller(config)
```

#### Poll

Waits for operation to reach target state.

```go
func (p *Poller) Poll(ctx context.Context) error
```

**Parameters:**
- `ctx`: Context (can be cancelled)

**Returns:** `error` if operation fails, times out, or is cancelled

**Example:**
```go
if err := poller.Poll(ctx); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

---

## Retry Logic

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/retry`

### RetryConfig

Configuration for retry behavior.

```go
type RetryConfig struct {
    MaxAttempts     int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    Multiplier      float64
    ShouldRetry     func(error) bool
}
```

**Fields:**
- `MaxAttempts`: Maximum number of retry attempts
- `InitialDelay`: Initial delay before first retry
- `MaxDelay`: Maximum delay between retries
- `Multiplier`: Exponential backoff multiplier
- `ShouldRetry`: Function to determine if error is retryable

#### DefaultRetryConfig

Returns sensible default configuration.

```go
func DefaultRetryConfig() RetryConfig
```

**Returns:** `RetryConfig` with defaults:
- `MaxAttempts`: 3
- `InitialDelay`: 1 second
- `MaxDelay`: 30 seconds
- `Multiplier`: 2.0
- `ShouldRetry`: Always returns true

**Example:**
```go
config := retry.DefaultRetryConfig()
config.MaxAttempts = 5
config.ShouldRetry = func(err error) bool {
    if resErr, ok := err.(*errors.ResourceError); ok {
        return errors.IsRetryable(resErr.StatusCode)
    }
    return false
}
```

#### Retry

Executes operation with exponential backoff.

```go
func Retry(ctx context.Context, config RetryConfig, operation func() error) error
```

**Parameters:**
- `ctx`: Context (can be cancelled)
- `config`: Retry configuration
- `operation`: Function to retry

**Returns:** `error` if all attempts fail or operation is cancelled

**Example:**
```go
err := retry.Retry(ctx, retry.DefaultRetryConfig(), func() error {
    _, _, err := client.CreateVolume(auth).Execute()
    return err
})
```

---

## Logging

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/logging`

### Logger

Structured logger with sensitive data sanitization.

```go
type Logger struct {
    ctx context.Context
}
```

#### NewLogger

Creates a new Logger.

```go
func NewLogger(ctx context.Context) *Logger
```

**Parameters:**
- `ctx`: Context for logging

**Returns:** `*Logger`

**Example:**
```go
logger := logging.NewLogger(ctx)
```

#### LogAPIRequest

Logs an API request with automatic sanitization.

```go
func (l *Logger) LogAPIRequest(method, path string, body interface{})
```

**Parameters:**
- `method`: HTTP method (GET, POST, PUT, DELETE)
- `path`: API endpoint path
- `body`: Request body (will be sanitized)

**Example:**
```go
logger.LogAPIRequest("POST", "/volumes", volumeRequest)
```

#### LogAPIResponse

Logs an API response.

```go
func (l *Logger) LogAPIResponse(method, path string, statusCode int, body interface{})
```

**Parameters:**
- `method`: HTTP method
- `path`: API endpoint path
- `statusCode`: HTTP status code
- `body`: Response body

**Example:**
```go
logger.LogAPIResponse("POST", "/volumes", 201, volumeResponse)
```

#### LogStateTransition

Logs a resource state transition.

```go
func (l *Logger) LogStateTransition(resourceType, resourceID, oldState, newState string)
```

**Parameters:**
- `resourceType`: Terraform resource type
- `resourceID`: Resource identifier
- `oldState`: Previous state
- `newState`: New state

**Example:**
```go
logger.LogStateTransition("emma_volume", "12345", "creating", "available")
```

#### LogError

Logs an error with context.

```go
func (l *Logger) LogError(message string, fields map[string]interface{})
```

**Parameters:**
- `message`: Error message
- `fields`: Additional context fields

**Example:**
```go
logger.LogError("Failed to create volume", map[string]interface{}{
    "volume_id": volumeID,
    "error":     err.Error(),
})
```

#### SanitizeValue

Removes sensitive data from any value.

```go
func SanitizeValue(value interface{}) interface{}
```

**Parameters:**
- `value`: Value to sanitize

**Returns:** Sanitized value with sensitive fields redacted

**Sensitive Fields:**
- `password`, `token`, `secret`, `key`, `credential`, `auth`

**Example:**
```go
sanitized := logging.SanitizeValue(data)
```

---

## Validation

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/validation`

### MutuallyExclusive

Validator that ensures only one field in a group is set.

```go
type MutuallyExclusive struct {
    Fields []string
}
```

**Usage:**
```go
"volume_type": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        validation.MutuallyExclusive([]string{"volume_type", "volume_config_id"}),
    },
}
```

### RequiresOneOf

Validator that ensures at least one field in a group is set.

```go
type RequiresOneOf struct {
    Fields []string
}
```

**Usage:**
```go
"instance_id": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        validation.RequiresOneOf([]string{"instance_id", "spot_instance_id"}),
    },
}
```

---

## Testing Utilities

Package: `github.com/emma-community/terraform-provider-emma/internal/emma/common/testing/fixtures`

### Test Fixtures

#### VolumeFixture

Returns consistent test data for volumes.

```go
func VolumeFixture() *emmaSdk.Volume
```

**Returns:** `*emmaSdk.Volume` with test data

**Example:**
```go
volume := fixtures.VolumeFixture()
```

#### VmFixture

Returns consistent test data for VMs.

```go
func VmFixture() *emmaSdk.Vm
```

**Returns:** `*emmaSdk.Vm` with test data

**Example:**
```go
vm := fixtures.VmFixture()
```

#### SshKeyFixture

Returns consistent test data for SSH keys.

```go
func SshKeyFixture() *emmaSdk.SshKey
```

**Returns:** `*emmaSdk.SshKey` with test data

**Example:**
```go
sshKey := fixtures.SshKeyFixture()
```

#### SecurityGroupFixture

Returns consistent test data for security groups.

```go
func SecurityGroupFixture() *emmaSdk.SecurityGroup
```

**Returns:** `*emmaSdk.SecurityGroup` with test data

**Example:**
```go
sg := fixtures.SecurityGroupFixture()
```

### Property Test Generators

#### VolumeConfigGen

Generates random valid volume configurations.

```go
func VolumeConfigGen() gopter.Gen
```

**Returns:** `gopter.Gen` that generates volume configs

**Example:**
```go
properties.Property("volumes are valid", prop.ForAll(
    func(config map[string]interface{}) bool {
        // Test with config
        return true
    },
    fixtures.VolumeConfigGen(),
))
```

#### InvalidVolumeConfigGen

Generates random invalid volume configurations.

```go
func InvalidVolumeConfigGen() gopter.Gen
```

**Returns:** `gopter.Gen` that generates invalid volume configs

**Example:**
```go
properties.Property("invalid volumes fail", prop.ForAll(
    func(config map[string]interface{}) bool {
        // Test with invalid config
        return true
    },
    fixtures.InvalidVolumeConfigGen(),
))
```

---

## Best Practices

### Error Handling

1. Always use `ErrorBuilder` for consistent error messages
2. Include resource type, operation, and ID in errors
3. Use `MapHTTPError()` for user-friendly messages
4. Check `IsRetryable()` before implementing retry logic

### Type Conversion

1. Use shared conversion utilities instead of manual conversions
2. Always check errors from conversion functions
3. Use null-safe functions for optional values
4. Use bounds-checking functions for numeric conversions

### State Management

1. Always use `StateManager.RemoveFromState()` for 404 responses
2. Use `PreserveUserValues()` to maintain user-specified values
3. Use `UpdateComputedAttributes()` for API-computed fields
4. Use `DriftDetector` to warn about external changes

### Async Operations

1. Always configure reasonable timeouts
2. Define both target and failure states
3. Use appropriate poll intervals (5-10 seconds typical)
4. Handle context cancellation

### Retry Logic

1. Start with `DefaultRetryConfig()` and customize as needed
2. Only retry transient errors (use `IsRetryable()`)
3. Set reasonable max attempts (3-5 typical)
4. Log retry attempts for debugging

### Logging

1. Use appropriate log levels (Debug, Info, Error)
2. Always use logging utilities for sensitive data
3. Include resource context in logs
4. Use structured logging for machine parsing

### Testing

1. Use fixtures for consistent test data
2. Write property-based tests for universal properties
3. Use generators for diverse test inputs
4. Test error paths and edge cases

---

## Version History

- **v1.0.0**: Initial release with all shared utilities
- Documentation reflects current implementation

## Support

For questions or issues with the API:
- Review the [Architecture Guide](./ARCHITECTURE.md)
- Check the [Testing Guide](./TESTING_GUIDE.md)
- Consult existing resource implementations
- Open an issue for clarification
