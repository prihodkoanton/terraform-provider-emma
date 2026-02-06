# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Async Operations and State Transitions**: Comprehensive state management for handling resource transitions
  - Automatic waiting for resources to reach stable states before operations
  - State transition manager for VMs (POWERED_ON/POWERED_OFF), Volumes (AVAILABLE), and Security Groups (RECOMPOSED)
  - Automatic retry for state conflict errors (409) with exponential backoff
  - Configurable timeouts via Terraform `timeouts` block for state polling
  - Enhanced error messages that include current and expected states
  - Idempotent state checks (operations succeed when already in target state)
  - Support for parallel operations without state interference
  - Graceful degradation with clear timeout errors for stuck resources

### Changed

- **VM Resource**: Now automatically waits for stable state before hardware edits, volume operations, and security group changes
- **Volume Resource**: Now automatically waits for stable state before attach/detach/resize operations, preserves `attached_to_id` during transitions
- **Security Group Resource**: Now automatically waits for RECOMPOSED state before and after updates, handles multiple sequential recompositions
- **Retry Logic**: Enhanced to detect and handle state conflict errors automatically
- **Error Messages**: State-related errors now include current state, expected state, and timeout information

### Fixed

- Volume attachment failures when VM is in BUSY state (now waits for stable state)
- Security group update failures during recomposition (now waits for RECOMPOSED state)
- "Resource conflict" errors during concurrent operations (now handles state transitions)
- Timeout errors now provide clear guidance on increasing timeout configuration

### Deprecated
### Removed
### Security

## [0.1.0] - 2026-02-04

FEATURES:

* **New Common Utilities Package**: Added `internal/emma/common/` with reusable utilities for error handling, type conversion, state management, async operations, and retry logic
* **Import Support**: Added import functionality for `emma_spot_instance` and `emma_kubernetes_cluster` resources, enabling management of existing infrastructure
* **Enhanced Logging**: Implemented structured logging throughout the provider with automatic sensitive data sanitization
* **Property-Based Testing**: Added comprehensive property-based tests using gopter to validate correctness properties across all utilities
* **Test Fixtures and Generators**: New testing utilities in `internal/emma/common/testing/` for consistent test data generation

IMPROVEMENTS:

* **Centralized Error Handling**: All resources now use consistent error handling with context-rich error messages including resource type, operation, and ID
* **HTTP Error Mapping**: User-friendly error messages for all HTTP status codes (400, 401, 403, 404, 409, 422, 429, 500, 503)
* **Type Conversion Utilities**: Shared conversion functions eliminate code duplication and ensure consistent null/unknown value handling
* **Enhanced Validation Framework**: New cross-field validators (`MutuallyExclusive`, `RequiresOneOf`) for complex validation scenarios
* **State Management Helpers**: Consistent patterns for state operations, 404 handling, computed attribute updates, and user value preservation
* **Drift Detection**: Automatic detection and reporting of differences between Terraform state and actual infrastructure
* **Async Operation Handling**: Configurable polling mechanism with timeout support for long-running operations (VM hardware edits, volume resizes, security group synchronization)
* **Retry Logic with Exponential Backoff**: Automatic retry for transient failures (429, 5xx errors) with configurable max attempts and delays
* **Provider Configuration**: Added retry configuration options (`max_retries`, `retry_delay`, `max_retry_delay`) to provider schema
* **Code Coverage**: Achieved >80% code coverage across all new utilities and migrated resources
* **Documentation**: Comprehensive developer and user documentation including architecture guide, API reference, testing guide, error handling guide, troubleshooting guide, and migration guide

ENHANCEMENTS:

* **Volume Resource**: Migrated to use centralized error handling, type conversion utilities, and state management helpers
* **VM Resource**: Updated to use new async polling mechanism for hardware edits and volume resize operations
* **SSH Key Resource**: Migrated to use centralized error handling and type conversion utilities
* **Security Group Resource**: Updated to use configurable async poller for synchronization and recomposing status
* **All Resources**: Consistent error messages, improved state management, and better handling of edge cases

TESTING:

* Added 12 correctness properties validated through property-based tests
* Integration tests for all migrated resources (volume, VM, SSH key, security group)
* Unit tests for all utility functions with >80% coverage
* Property tests for error context, type conversions, null handling, retry logic, exponential backoff, async timeouts, sensitive data logging, and import completeness

DOCUMENTATION:

* `docs/ARCHITECTURE.md`: Overview of provider architecture and new utilities
* `docs/API_REFERENCE.md`: Complete API reference for all exported functions and types
* `docs/TESTING_GUIDE.md`: Guide for writing property-based and integration tests
* `docs/ERROR_HANDLING.md`: Common errors, solutions, and retry behavior
* `docs/TROUBLESHOOTING.md`: Debug logging, error interpretation, and common issues
* `docs/MIGRATION.md`: Upgrade instructions and breaking changes (none in this release)
* `docs/LOGGING.md`: Logging configuration and sensitive data handling
* Updated resource documentation with import examples

NOTES:

* All improvements maintain backward compatibility - no breaking changes
* Existing Terraform configurations continue to work without modification
* New utilities are available for use but existing code paths remain functional
* Migration to new patterns was done incrementally to ensure stability

## Version History

[Unreleased]: https://github.com/emma-community/terraform-provider-emma/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/emma-community/terraform-provider-emma/releases/tag/v0.1.0
