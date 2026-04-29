# Emma Terraform Provider - Testing Guide

## Overview

This guide covers testing strategies and best practices for the Emma Terraform Provider. We use a combination of unit tests, property-based tests, and integration tests to ensure comprehensive coverage and correctness.

## Table of Contents

- [Testing Philosophy](#testing-philosophy)
- [Test Types](#test-types)
- [Property-Based Testing](#property-based-testing)
- [Unit Testing](#unit-testing)
- [Integration Testing](#integration-testing)
- [Test Fixtures and Generators](#test-fixtures-and-generators)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Best Practices](#best-practices)

---

## Testing Philosophy

Our testing approach is based on these principles:

1. **Correctness First**: Property-based tests validate universal properties
2. **Comprehensive Coverage**: Aim for >80% code coverage
3. **Fast Feedback**: Unit tests run quickly for rapid iteration
4. **Real-World Validation**: Integration tests validate against actual API
5. **Maintainability**: Use fixtures and generators for consistent test data

### Test Pyramid

```
        /\
       /  \      Integration Tests (Few, Slow, High Confidence)
      /____\
     /      \    Property-Based Tests (Some, Medium, High Coverage)
    /________\
   /          \  Unit Tests (Many, Fast, Specific Cases)
  /____________\
```

---

## Test Types

### Unit Tests

**Purpose**: Test individual functions and methods in isolation

**Characteristics**:
- Fast execution (milliseconds)
- Test specific examples and edge cases
- Mock external dependencies
- Focus on one function/method at a time

**When to Use**:
- Testing error handling paths
- Testing specific edge cases
- Testing helper functions
- Testing validation logic

### Property-Based Tests

**Purpose**: Validate universal properties across many generated inputs

**Characteristics**:
- Medium execution time (seconds)
- Test properties that should hold for all inputs
- Generate diverse test cases automatically
- Catch edge cases you didn't think of

**When to Use**:
- Testing conversion functions (round-trip properties)
- Testing validation logic (all invalid inputs rejected)
- Testing state management (user values preserved)
- Testing retry logic (max attempts respected)

### Integration Tests

**Purpose**: Validate end-to-end workflows with real API

**Characteristics**:
- Slow execution (minutes)
- Test complete CRUD operations
- Use real API (requires credentials)
- Validate actual behavior

**When to Use**:
- Testing resource lifecycle (Create, Read, Update, Delete)
- Testing async operations
- Testing error handling with real API responses
- Validating migrations to new utilities

---

## Property-Based Testing

Property-based testing is a powerful technique for validating correctness. Instead of writing specific test cases, you define properties that should hold for all inputs.

### What is a Property?

A property is a statement that should be true for all valid inputs. Properties are universally quantified:

- "For any valid volume configuration, creating then reading should return the same values"
- "For any integer, converting to string and back should preserve the value"
- "For any retry configuration, the number of attempts should not exceed max attempts"

### Using gopter

We use [gopter](https://github.com/leanovate/gopter) for property-based testing in Go.

#### Basic Structure

```go
import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/prop"
)

func TestPropertyExample(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("description of property", prop.ForAll(
        func(input InputType) bool {
            // Test the property
            result := functionUnderTest(input)
            return result == expectedCondition
        },
        generatorForInput(),
    ))
    
    properties.TestingRun(t)
}
```

#### Example: Round-Trip Property

```go
func TestTypeConversionRoundTrip(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("int32 to string and back preserves value", prop.ForAll(
        func(value int32) bool {
            // Convert to string
            tfString := convert.Int32ToString(&value)
            
            // Convert back to int32
            result, err := convert.StringToInt32(tfString)
            
            // Should preserve value
            return err == nil && result == value
        },
        gen.Int32(),
    ))
    
    properties.TestingRun(t)
}
```

#### Example: Error Handling Property

```go
func TestRetryRespectsMaxAttempts(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("retry respects max attempts", prop.ForAll(
        func(maxAttempts int) bool {
            if maxAttempts < 1 || maxAttempts > 10 {
                return true // Skip invalid configs
            }
            
            attempts := 0
            config := retry.RetryConfig{
                MaxAttempts: maxAttempts,
                InitialDelay: 1 * time.Millisecond,
                MaxDelay: 10 * time.Millisecond,
                Multiplier: 2.0,
                ShouldRetry: func(error) bool { return true },
            }
            
            retry.Retry(context.Background(), config, func() error {
                attempts++
                return fmt.Errorf("always fails")
            })
            
            return attempts == maxAttempts
        },
        gen.IntRange(1, 10),
    ))
    
    properties.TestingRun(t)
}
```

#### Example: State Transition Property

```go
// Feature: async-operations, Property 1: State Polling Eventually Terminates
// Validates: Requirements 1.5, 4.4
func TestPropertyStatePollingTerminates(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("state polling terminates within timeout", prop.ForAll(
        func(timeoutMs, pollIntervalMs int) bool {
            if timeoutMs < pollIntervalMs || pollIntervalMs < 10 {
                return true // Skip invalid configs
            }
            
            timeout := time.Duration(timeoutMs) * time.Millisecond
            pollInterval := time.Duration(pollIntervalMs) * time.Millisecond
            
            callCount := 0
            startTime := time.Now()
            
            config := async.PollerConfig{
                Timeout:      timeout,
                PollInterval: pollInterval,
                StatusChecker: func(ctx context.Context) (string, error) {
                    callCount++
                    // Never reach target state to test timeout
                    return "creating", nil
                },
                TargetStates:  []string{"available"},
                FailureStates: []string{"error"},
            }
            
            poller := async.NewPoller(config)
            err := poller.Poll(context.Background())
            
            duration := time.Since(startTime)
            
            // Should timeout with error
            // Duration should be close to timeout (within 20% tolerance)
            return err != nil && 
                   duration >= timeout && 
                   duration < timeout*12/10
        },
        gen.IntRange(100, 1000),   // timeout: 100-1000ms
        gen.IntRange(10, 100),     // poll interval: 10-100ms
    ))
    
    properties.TestingRun(t)
}
```

#### Example: Idempotent State Check Property

```go
// Feature: async-operations, Property 8: Idempotent State Checks
// Validates: Requirements 8.1, 8.4
func TestPropertyIdempotentStateChecks(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("operations succeed when already in target state", prop.ForAll(
        func(targetState string) bool {
            // Resource already in target state
            config := state.StateTransitionConfig{
                ResourceType: "vm",
                ResourceID:   "test-vm",
                StatusChecker: func(ctx context.Context) (string, error) {
                    return targetState, nil
                },
                TargetStates:       []string{targetState},
                TransitionalStates: []string{"BUSY"},
                FailureStates:      []string{"error"},
                Timeout:            1 * time.Second,
                PollInterval:       100 * time.Millisecond,
            }
            
            manager := state.NewStateTransitionManager(config)
            err := manager.WaitForStableState(context.Background())
            
            // Should succeed immediately without error
            return err == nil
        },
        gen.OneConstOf("POWERED_ON", "POWERED_OFF", "AVAILABLE", "RECOMPOSED"),
    ))
    
    properties.TestingRun(t)
}
```

### Writing Generators

Generators create random test inputs. Write smart generators that constrain to valid input space.

#### Simple Generators

```go
import "github.com/leanovate/gopter/gen"

// Generate integers in range
gen.IntRange(1, 100)

// Generate strings
gen.Identifier()  // Valid identifiers
gen.AlphaString() // Alphabetic strings

// Generate booleans
gen.Bool()

// Generate from constants
gen.OneConstOf("ssd", "ssd-plus", "hdd")
```

#### Complex Generators

```go
// Generate volume configurations
func VolumeConfigGen() gopter.Gen {
    return gopter.CombineGens(
        gen.Identifier(),                    // name
        gen.Identifier(),                    // data_center_id
        gen.IntRange(1, 10000),             // volume_gb
        gen.OneConstOf("ssd", "ssd-plus"),  // volume_type
    ).Map(func(values []interface{}) map[string]interface{} {
        return map[string]interface{}{
            "name":           values[0].(string),
            "data_center_id": values[1].(string),
            "volume_gb":      values[2].(int),
            "volume_type":    values[3].(string),
        }
    })
}
```

#### Invalid Input Generators

```go
// Generate invalid configurations for testing validation
func InvalidVolumeConfigGen() gopter.Gen {
    return gen.OneGenOf(
        // Missing required fields
        gen.Const(map[string]interface{}{
            "name": "test",
        }),
        // Invalid volume size
        gen.Const(map[string]interface{}{
            "name":           "test",
            "data_center_id": "dc-1",
            "volume_gb":      -1,
            "volume_type":    "ssd",
        }),
        // Empty volume type
        gen.Const(map[string]interface{}{
            "name":           "test",
            "data_center_id": "dc-1",
            "volume_gb":      100,
            "volume_type":    "",
        }),
    )
}
```

### Property Test Configuration

Configure property tests to run enough iterations:

```go
properties := gopter.NewProperties(&gopter.TestParameters{
    MinSuccessfulTests: 100,  // Run at least 100 test cases
    MaxSize:            1000, // Maximum size for generated values
    Rng:                rand.New(rand.NewSource(0)), // Deterministic seed
})
```

### Tagging Property Tests

Tag each property test with the feature and property number:

```go
// Feature: provider-improvements, Property 2: Type Conversions Preserve Values
func TestTypeConversionRoundTrip(t *testing.T) {
    // Test implementation
}
```

---

## Unit Testing

Unit tests validate specific examples and edge cases.

### Structure

```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := setupTestInput()
    
    // Act
    result := functionUnderTest(input)
    
    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Using testify

We use [testify](https://github.com/stretchr/testify) for assertions:

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    result, err := functionUnderTest()
    
    // require stops test on failure
    require.NoError(t, err)
    
    // assert continues test on failure
    assert.Equal(t, expected, result)
    assert.NotNil(t, result)
    assert.True(t, condition)
}
```

### Table-Driven Tests

Use table-driven tests for multiple similar cases:

```go
func TestMapHTTPError(t *testing.T) {
    tests := []struct {
        name       string
        statusCode int
        apiMessage string
        want       string
    }{
        {
            name:       "404 not found",
            statusCode: 404,
            apiMessage: "Volume not found",
            want:       "Resource not found. It may have been deleted.",
        },
        {
            name:       "500 server error",
            statusCode: 500,
            apiMessage: "Internal error",
            want:       "Server error. Please try again or contact support.",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := errors.MapHTTPError(tt.statusCode, tt.apiMessage)
            assert.Contains(t, got, tt.want)
        })
    }
}
```

### Testing Error Cases

Always test error paths:

```go
func TestStringToInt32_InvalidInput(t *testing.T) {
    tests := []struct {
        name  string
        input types.String
    }{
        {"null value", types.StringNull()},
        {"unknown value", types.StringUnknown()},
        {"invalid format", types.StringValue("not-a-number")},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := convert.StringToInt32(tt.input)
            assert.Error(t, err)
        })
    }
}
```

---

## Integration Testing

Integration tests validate complete workflows with the real API.

### Setup

Integration tests require:
- `TF_ACC=1` environment variable
- Valid Emma credentials (`EMMA_CLIENT_ID`, `EMMA_CLIENT_SECRET`)
- Access to Emma API

### Structure

```go
func TestVolumeResource_Integration(t *testing.T) {
    if os.Getenv("TF_ACC") != "1" {
        t.Skip("Skipping integration test")
    }
    
    // Setup
    client := setupTestClient(t)
    
    // Test Create
    volume := createTestVolume(t, client)
    defer cleanupVolume(t, client, volume.Id)
    
    // Test Read
    readVolume := readTestVolume(t, client, volume.Id)
    assert.Equal(t, volume.Name, readVolume.Name)
    
    // Test Update
    updateTestVolume(t, client, volume.Id)
    
    // Test Delete
    deleteTestVolume(t, client, volume.Id)
}
```

### Best Practices

1. **Always cleanup**: Use `defer` to cleanup resources
2. **Use unique names**: Avoid conflicts with other tests
3. **Test error cases**: Test 404, validation errors, etc.
4. **Be patient**: Use appropriate timeouts for async operations
5. **Isolate tests**: Each test should be independent

### Example Integration Test

```go
func TestVolumeCreate_Integration(t *testing.T) {
    if os.Getenv("TF_ACC") != "1" {
        t.Skip("Skipping integration test")
    }
    
    client := setupTestClient(t)
    ctx := context.Background()
    
    // Create volume
    volumeName := fmt.Sprintf("test-volume-%d", time.Now().Unix())
    volumeRequest := emmaSdk.VolumeCreate{
        Name:         &volumeName,
        DataCenterId: ptr("dc-1"),
        VolumeGb:     ptr(int32(100)),
        VolumeType:   ptr("ssd"),
    }
    
    volume, resp, err := client.VolumesAPI.CreateVolume(ctx).
        VolumeCreate(volumeRequest).
        Execute()
    
    require.NoError(t, err)
    require.Equal(t, 201, resp.StatusCode)
    require.NotNil(t, volume.Id)
    
    // Cleanup
    defer func() {
        client.VolumesAPI.DeleteVolume(ctx, *volume.Id).Execute()
    }()
    
    // Wait for available
    poller := async.NewPoller(async.PollerConfig{
        Timeout:      5 * time.Minute,
        PollInterval: 10 * time.Second,
        StatusChecker: func(ctx context.Context) (string, error) {
            v, _, err := client.VolumesAPI.GetVolume(ctx, *volume.Id).Execute()
            if err != nil {
                return "", err
            }
            return *v.Status, nil
        },
        TargetStates:  []string{"available"},
        FailureStates: []string{"error"},
    })
    
    err = poller.Poll(ctx)
    require.NoError(t, err)
    
    // Verify volume
    readVolume, _, err := client.VolumesAPI.GetVolume(ctx, *volume.Id).Execute()
    require.NoError(t, err)
    assert.Equal(t, volumeName, *readVolume.Name)
    assert.Equal(t, "available", *readVolume.Status)
}
```

---

## Test Fixtures and Generators

### Using Fixtures

Fixtures provide consistent test data:

```go
import "github.com/emma-community/terraform-provider-emma/internal/emma/common/testing/fixtures"

func TestVolumeConversion(t *testing.T) {
    // Get consistent test volume
    volume := fixtures.VolumeFixture()
    
    // Use in test
    state := convertVolumeToState(volume)
    assert.Equal(t, *volume.Name, state.Name.ValueString())
}
```

### Available Fixtures

- `VolumeFixture()`: Test volume data
- `VmFixture()`: Test VM data
- `SshKeyFixture()`: Test SSH key data
- `SecurityGroupFixture()`: Test security group data

### Using Generators

Generators create diverse test inputs for property tests:

```go
func TestVolumeValidation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("valid volumes pass validation", prop.ForAll(
        func(config map[string]interface{}) bool {
            err := validateVolumeConfig(config)
            return err == nil
        },
        fixtures.VolumeConfigGen(),
    ))
    
    properties.Property("invalid volumes fail validation", prop.ForAll(
        func(config map[string]interface{}) bool {
            err := validateVolumeConfig(config)
            return err != nil
        },
        fixtures.InvalidVolumeConfigGen(),
    ))
    
    properties.TestingRun(t)
}
```

### Creating Custom Generators

```go
// Generate VM configurations
func VmConfigGen() gopter.Gen {
    return gopter.CombineGens(
        gen.Identifier(),           // name
        gen.IntRange(1, 32),       // vcpu
        gen.IntRange(1, 256),      // ram_gb
        gen.Identifier(),          // os_id
        gen.Identifier(),          // data_center_id
    ).Map(func(values []interface{}) map[string]interface{} {
        return map[string]interface{}{
            "name":           values[0].(string),
            "vcpu":           values[1].(int),
            "ram_gb":         values[2].(int),
            "os_id":          values[3].(string),
            "data_center_id": values[4].(string),
        }
    })
}
```

---

## Running Tests

### Unit Tests

Run all unit tests:

```bash
go test ./internal/emma/... -v
```

Run tests for specific package:

```bash
go test ./internal/emma/common/errors/... -v
```

Run specific test:

```bash
go test ./internal/emma/common/errors/... -v -run TestMapHTTPError
```

### Property-Based Tests

Property tests run as part of unit tests:

```bash
go test ./internal/emma/common/convert/... -v -run TestProperty
```

### Integration Tests

Run integration tests (requires credentials):

```bash
TF_ACC=1 go test ./internal/emma/... -v -timeout 120m
```

Run specific integration test:

```bash
TF_ACC=1 go test ./internal/emma/... -v -run TestVolumeResource_Integration
```

### Coverage

Generate coverage report:

```bash
go test ./internal/emma/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Check coverage percentage:

```bash
go test ./internal/emma/... -cover
```

### Makefile Commands

Use makefile for common tasks:

```bash
# Run acceptance tests
make testacc

# Run unit tests
make test

# Generate documentation
make docs
```

---

## Writing Tests

### Test File Naming

- Unit tests: `*_test.go` (e.g., `errors_test.go`)
- Property tests: `*_property_test.go` (e.g., `convert_property_test.go`)
- Integration tests: `*_integration_test.go` (e.g., `volume_resource_integration_test.go`)

### Test Function Naming

- Unit tests: `TestFunctionName` (e.g., `TestMapHTTPError`)
- Property tests: `TestPropertyDescription` (e.g., `TestPropertyRoundTrip`)
- Integration tests: `TestResourceName_Integration` (e.g., `TestVolumeResource_Integration`)

### Test Organization

```go
// Unit test file: errors_test.go
package errors

import "testing"

func TestMapHTTPError(t *testing.T) {
    // Test implementation
}

func TestIsRetryable(t *testing.T) {
    // Test implementation
}
```

```go
// Property test file: errors_property_test.go
package errors

import (
    "testing"
    "github.com/leanovate/gopter"
)

// Feature: provider-improvements, Property 1: Error Messages Include Context
func TestPropertyErrorContext(t *testing.T) {
    // Property test implementation
}
```

### Writing a Complete Test Suite

Example for a new utility:

```go
// 1. Unit tests (basic_test.go)
func TestBasicFunctionality(t *testing.T) {
    // Test specific examples
}

func TestErrorCases(t *testing.T) {
    // Test error handling
}

func TestEdgeCases(t *testing.T) {
    // Test boundary conditions
}

// 2. Property tests (basic_property_test.go)
func TestPropertyRoundTrip(t *testing.T) {
    // Test universal properties
}

// 3. Integration tests (basic_integration_test.go)
func TestBasic_Integration(t *testing.T) {
    if os.Getenv("TF_ACC") != "1" {
        t.Skip("Skipping integration test")
    }
    // Test with real API
}
```

---

## Best Practices

### General

1. **Test behavior, not implementation**: Focus on what the code does, not how
2. **One assertion per test**: Keep tests focused and clear
3. **Use descriptive names**: Test names should explain what is being tested
4. **Keep tests simple**: Tests should be easier to understand than the code
5. **Test error paths**: Don't just test the happy path

### Property-Based Testing

1. **Write smart generators**: Constrain to valid input space
2. **Run enough iterations**: Minimum 100 iterations per property
3. **Test universal properties**: Properties that hold for all inputs
4. **Tag with property number**: Reference design document properties
5. **Handle edge cases in generators**: Include boundary values

### Unit Testing

1. **Use table-driven tests**: For multiple similar cases
2. **Test one thing**: Each test should verify one behavior
3. **Use fixtures**: For consistent test data
4. **Mock external dependencies**: Keep tests fast and isolated
5. **Test error messages**: Verify error messages are helpful

### Integration Testing

1. **Always cleanup**: Use defer to cleanup resources
2. **Use unique names**: Avoid conflicts between tests
3. **Be patient**: Use appropriate timeouts
4. **Test real scenarios**: Test actual user workflows
5. **Isolate tests**: Each test should be independent

### Code Coverage

1. **Aim for >80%**: But don't obsess over 100%
2. **Focus on critical paths**: Prioritize important code
3. **Test error paths**: Often missed in coverage
4. **Use coverage tools**: Identify untested code
5. **Don't game metrics**: Write meaningful tests

### Test Maintenance

1. **Keep tests up to date**: Update tests when code changes
2. **Remove obsolete tests**: Delete tests for removed features
3. **Refactor tests**: Apply same standards as production code
4. **Document complex tests**: Explain non-obvious test logic
5. **Review test failures**: Don't ignore flaky tests

---

## Common Patterns

### Testing Error Handling

```go
func TestErrorHandling(t *testing.T) {
    err := errors.NewError("emma_volume", "Create").
        WithID("12345").
        WithStatusCode(404).
        WithMessage("Volume not found").
        Build()
    
    assert.Contains(t, err.Error(), "emma_volume")
    assert.Contains(t, err.Error(), "Create")
    assert.Contains(t, err.Error(), "Volume not found")
}
```

### Testing Type Conversions

```go
func TestTypeConversion(t *testing.T) {
    // Test valid conversion
    value := int32(100)
    tfString := convert.Int32ToString(&value)
    assert.Equal(t, "100", tfString.ValueString())
    
    // Test null handling
    tfNull := convert.Int32ToString(nil)
    assert.True(t, tfNull.IsNull())
}
```

### Testing State Management

```go
func TestStateManagement(t *testing.T) {
    sm := state.NewStateManager(context.Background())
    
    // Test computed attribute update
    currentState := &VolumeState{Name: "test", Status: "creating"}
    apiResponse := &Volume{Name: "test", Status: "available"}
    
    err := sm.UpdateComputedAttributes(currentState, apiResponse, []string{"Status"})
    assert.NoError(t, err)
    assert.Equal(t, "available", currentState.Status)
}
```

### Testing Async Operations

```go
func TestAsyncOperation(t *testing.T) {
    callCount := 0
    config := async.PollerConfig{
        Timeout:      1 * time.Second,
        PollInterval: 100 * time.Millisecond,
        StatusChecker: func(ctx context.Context) (string, error) {
            callCount++
            if callCount >= 3 {
                return "available", nil
            }
            return "creating", nil
        },
        TargetStates:  []string{"available"},
        FailureStates: []string{"error"},
    }
    
    poller := async.NewPoller(config)
    err := poller.Poll(context.Background())
    
    assert.NoError(t, err)
    assert.GreaterOrEqual(t, callCount, 3)
}
```

### Testing State Transitions

```go
func TestStateTransitionManager(t *testing.T) {
    // Test waiting for stable state
    currentState := "BUSY"
    config := state.StateTransitionConfig{
        ResourceType: "vm",
        ResourceID:   "test-vm-123",
        StatusChecker: func(ctx context.Context) (string, error) {
            // Simulate state transition
            if currentState == "BUSY" {
                currentState = "POWERED_ON"
            }
            return currentState, nil
        },
        TargetStates:       []string{"POWERED_ON", "POWERED_OFF"},
        TransitionalStates: []string{"BUSY"},
        FailureStates:      []string{"error"},
        Timeout:            5 * time.Second,
        PollInterval:       100 * time.Millisecond,
    }
    
    manager := state.NewStateTransitionManager(config)
    err := manager.WaitForStableState(context.Background())
    
    assert.NoError(t, err)
    assert.Equal(t, "POWERED_ON", currentState)
}
```

### Testing State Conflict Retry

```go
func TestStateConflictRetry(t *testing.T) {
    attempts := 0
    resourceState := "BUSY"
    
    config := retry.StateConflictRetryConfig()
    config.MaxAttempts = 3
    config.InitialDelay = 1 * time.Millisecond
    config.ShouldRetry = func(err error) bool {
        return retry.IsStateConflictError(err, 409, "resource is busy")
    }
    
    err := retry.Retry(context.Background(), config, func() error {
        attempts++
        if resourceState == "BUSY" {
            if attempts >= 2 {
                resourceState = "POWERED_ON"
            }
            return fmt.Errorf("resource is busy")
        }
        return nil
    })
    
    assert.NoError(t, err)
    assert.Equal(t, 2, attempts)
    assert.Equal(t, "POWERED_ON", resourceState)
}
```

### Testing Retry Logic

```go
func TestRetryLogic(t *testing.T) {
    attempts := 0
    config := retry.DefaultRetryConfig()
    config.MaxAttempts = 3
    config.InitialDelay = 1 * time.Millisecond
    
    err := retry.Retry(context.Background(), config, func() error {
        attempts++
        if attempts < 3 {
            return fmt.Errorf("transient error")
        }
        return nil
    })
    
    assert.NoError(t, err)
    assert.Equal(t, 3, attempts)
}
```

---

## Troubleshooting

### Tests Fail Intermittently

**Cause**: Race conditions or timing issues

**Solution**:
- Use proper synchronization
- Increase timeouts for async operations
- Use deterministic random seeds for property tests

### Property Tests Take Too Long

**Cause**: Too many iterations or slow generators

**Solution**:
- Reduce `MinSuccessfulTests` during development
- Optimize generators
- Use faster test data

### Integration Tests Fail

**Cause**: API issues or missing credentials

**Solution**:
- Check credentials are set
- Verify API is accessible
- Check for rate limiting
- Review API error messages

### Low Code Coverage

**Cause**: Missing tests for error paths or edge cases

**Solution**:
- Use coverage tools to identify gaps
- Add tests for error handling
- Test edge cases and boundary conditions
- Add property tests for universal coverage

---

## Additional Resources

- [gopter Documentation](https://github.com/leanovate/gopter)
- [testify Documentation](https://github.com/stretchr/testify)
- [Terraform Plugin Testing](https://developer.hashicorp.com/terraform/plugin/testing)
- [Architecture Guide](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)

## Contributing

When adding tests:

1. Follow the patterns in this guide
2. Write property tests for universal properties
3. Write unit tests for specific cases
4. Write integration tests for critical workflows
5. Maintain >80% code coverage
6. Document complex test logic

## Questions and Support

For questions about testing:
- Review existing tests for examples
- Check the architecture guide for patterns
- Consult the API reference for utilities
- Open an issue for clarification
