# Testing Guidelines

This document outlines the testing practices for `truenas_incus_ctl`, including the red/green bug fixing methodology.

## Red/Green Bug Fixing Process

When fixing bugs, we follow a disciplined red/green testing approach to ensure we understand the problem, implement a proper test, and verify the fix works correctly.

### Process Overview

1. **🔴 Red Phase - Confirm the Bug**
   - Reproduce the bug in the current codebase
   - Understand the exact failure conditions
   - Document the expected vs actual behavior

2. **🔴 Red Phase - Implement Test (if practical)**
   - Write a test that exposes the bug
   - Run the test to confirm it fails (red light)
   - This test becomes our regression prevention

3. **🟢 Green Phase - Fix the Bug**
   - Implement the minimal fix needed
   - Run the test to confirm it passes (green light)
   - Verify no other tests are broken

4. **✅ Validate & Document**
   - Run full test suite
   - Document the fix and root cause
   - Commit with clear description

### When to Red/Green

**Always offer red/green testing when asked to fix a bug.**

The red/green approach should be used for:
- ✅ Reproducible bugs with clear failure conditions
- ✅ Logic errors, crashes, or incorrect behavior
- ✅ Environment-specific issues (like the HOME variable bug)
- ✅ Integration or system-level failures

Skip red/green for:
- ❌ Simple typos or obvious syntax errors
- ❌ Documentation-only changes
- ❌ Bugs that are impractical to test automatically

### Example: HOME Variable Bug Fix

This is exactly how we handled the recent HOME environment variable bug:

#### 🔴 Red Phase 1: Confirm Bug
```bash
# Reproduce the issue
env -u HOME ./truenas_incus_ctl config list
# Result: "2025/08/12 03:06:23 $HOME is not defined"
```

#### 🔴 Red Phase 2: Implement Test
Created `test/test_home_fallback.sh` that:
- Tests for "$HOME is not defined" crash
- Verifies config path fallback behavior
- Tests various HOME scenarios

Confirmed test failed with broken code:
```bash
./test/test_home_fallback.sh
# Result: "✗ FAIL: Config resolution failed without HOME"
```

#### 🟢 Green Phase: Fix Bug
- Implemented `getHomeDirWithFallback()` function
- Updated config and socket path resolution
- Confirmed test passes:
```bash
./test/test_home_fallback.sh  
# Result: "=== All HOME fallback tests passed! ==="
```

## Test Organization

### Unit Tests
- **Location**: `cmd/*_test.go`, `core/*_test.go`
- **Run**: `go test ./cmd ./core`
- **Purpose**: Test individual functions and components

### Integration Tests  
- **Location**: `test/*.sh`
- **Run**: `./run_tests.sh` or individually
- **Purpose**: Test end-to-end behavior, environment issues

### Build Tests
- **Location**: `build-install.sh`
- **Purpose**: Full build, test, and install validation

## Test Infrastructure

### Running Tests

```bash
# All tests (recommended)
./run_tests.sh

# Just Go unit tests
go test ./cmd ./core

# Just integration tests  
./test/test_home_fallback.sh

# Build with tests
./build-install.sh
```

### Adding New Tests

1. **Unit Tests**: Add `*_test.go` files in appropriate packages
2. **Integration Tests**: Add executable shell scripts in `test/` directory
3. **Update Runners**: Add new integration tests to `run_tests.sh`

## Red/Green Template

When asked to fix a bug, use this template:

```
I'll use the red/green approach to fix this bug:

**🔴 Red Phase**: First, let me reproduce the bug and confirm the issue
**🔴 Red Phase**: Then implement a test that exposes the problem (if practical)  
**🟢 Green Phase**: Finally, fix the bug and verify the test passes

Would you like me to proceed with red/green testing for this bug?
```

## Best Practices

- **Always reproduce first**: Don't fix what you can't reproduce
- **Test the fix, not just the feature**: Ensure your test actually catches the bug
- **Keep tests focused**: One test should verify one specific behavior
- **Make tests maintainable**: Use clear names and document complex test logic
- **Run full test suite**: Ensure fixes don't break existing functionality
- **Document root causes**: Help future developers understand the issue

## Benefits

The red/green approach provides:
- **Confidence**: We know the fix actually works
- **Regression prevention**: Tests catch if the bug returns
- **Documentation**: Tests serve as executable specifications
- **Debugging aid**: Clear reproduction steps for future investigation