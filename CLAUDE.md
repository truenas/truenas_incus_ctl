# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`truenas_incus_ctl` is a Go CLI tool for administering TrueNAS servers from Incus (LXD) containers. It provides dataset, snapshot, iSCSI, NFS, and replication management through TrueNAS API calls over WebSocket connections.

## Build & Development Commands

### Build
```bash
go build
```

### Install
```bash
go install
```

### Testing
```bash
# Run all tests
go test -v ./cmd

# Run specific test files
go test -v ./cmd/dataset_test.go ./cmd/util_testing.go
go test -v ./cmd/list_test.go ./cmd/util_testing.go
go test -v ./cmd/nfs_test.go ./cmd/util_testing.go
go test -v ./cmd/replication_test.go ./cmd/util_testing.go
go test -v ./cmd/snapshot_test.go ./cmd/util_testing.go
go test -v ./core/simple_queue_test.go
```

## Architecture

### Core Components

**cmd/** - Cobra CLI commands and business logic
- `root.go` - Main CLI setup, configuration management, daemon initialization
- `dataset.go`, `iscsi.go`, `nfs.go`, `replication.go`, `snapshot.go`, `share.go` - Feature-specific commands
- `list.go` - List operations for datasets, snapshots, shares
- `service.go` - TrueNAS service management
- `util_*.go` - Shared utilities and testing helpers

**core/** - Core networking and session management
- `daemon.go` - Connection-caching daemon with HTTP server over Unix sockets
- `client_api.go` - Client session implementation using the daemon
- `real_api.go` - Direct WebSocket session (bypass daemon)
- `session.go` - Abstract session interface
- `simple_queue.go` - Job queue implementation
- `future.go` - Future/promise implementation for async operations

**truenas_api/** - Pure TrueNAS WebSocket API client
- `truenas_api.go` - Low-level WebSocket client with job tracking

### Session Architecture

The tool supports two connection modes controlled by `USE_DAEMON` constant in `cmd/root.go`:

1. **Daemon Mode (default)**: Uses a connection-caching daemon (`core/daemon.go`) that runs as a background process, accessed via Unix socket. Multiple CLI invocations share WebSocket connections.

2. **Direct Mode**: Each CLI invocation creates its own WebSocket connection directly to TrueNAS.

### Configuration System

- Config file: `~/.truenas_incus_ctl/config.json`
- Supports multiple TrueNAS hosts with API keys
- Auto-selects first alphabetical host if none specified
- Command-line flags override config file settings

### Job Management

Long-running TrueNAS operations return job IDs. The system tracks these jobs through:
- WebSocket subscription to `core.get_jobs`
- Future-based async result handling
- Automatic job completion waiting on session close

## Key Patterns & Implementation Details

### Error Handling
- Extensive use of `fmt.Errorf()` for contextual error wrapping
- Job results parsed for both success data and error arrays
- Session cleanup waits for pending jobs unless explicitly skipped
- `core.ExtractApiError*()` functions for consistent API error formatting

### WebSocket Communication
- JSON-RPC 2.0 protocol
- Automatic reconnection on connection failures
- Connection sharing via daemon for efficiency
- TLS configuration supports self-signed certificates with `--allow-insecure`
- Future-based async handling with timeout support

### CLI Framework Patterns
- **Flag Processing**: `cmd/util_flags.go` provides sophisticated flag handling with enum validation, auxiliary flags, and snake_case/kebab-case normalization
- **Command Wrapping**: `WrapCommandFunc()` pattern ensures consistent session initialization and cleanup
- **Dynamic Command Generation**: iSCSI CRUD commands are dynamically generated from feature maps (see `iscsi_crud.go:AddIscsiCrudCommands`)

### Query System (`cmd/util_common.go`)
- **Unified Query API**: `QueryApi()` handles complex filtering, pagination, and property selection
- **Multi-type Queries**: Supports ID, name, path-based queries with automatic type detection
- **Recursive Queries**: Built-in support for dataset hierarchy traversal
- **Bulk Operations**: `MaybeBulkApiCall*()` functions optimize multiple API calls into single bulk operations

### iSCSI Implementation Patterns
- **Portal Management**: Auto-creates portals on demand with IPv6 bracket normalization
- **Target Naming**: Consistent hash-based naming for long volume paths to avoid length limits
- **Device Discovery**: `/dev/disk/by-path` parsing for active iSCSI shares
- **Service Validation**: Pre-flight checks for TrueNAS service status

### Type System & Data Processing
- **Generic Utilities**: `core/util.go` and `core/util_json.go` provide type-safe generics for common operations
- **Size String Parsing**: `ParseSizeString()` handles human-readable sizes (K/M/G/T/P with decimal support)
- **Property Mapping**: Dynamic property insertion with value ordering for parsed/raw/display values
- **Table Generation**: Flexible table formatting (CSV, JSON, table, compact) with column auto-discovery

### Testing Framework (`cmd/util_testing.go`)
- **Mock Session**: `UnitTestSession` for testing without live TrueNAS connection
- **Expected/Response Matching**: JSON-based request/response validation
- **Table Output Testing**: Validates formatted output matches expectations

### Concurrency & Threading
- **Future Implementation**: `core/future.go` provides thread-safe promise/future pattern with timeout support
- **Daemon Session Pooling**: Multiple concurrent sessions per host with automatic load balancing
- **File System Monitoring**: `inotify`-based file creation/deletion monitoring for iSCSI device tracking

## Development Notes

### Red/Green Bug Fixing Process

**When asked to fix a bug, always offer to use the red/green approach:**

1. **🔴 Red Phase - Confirm Bug**: Reproduce the issue in current codebase
2. **🔴 Red Phase - Implement Test**: Write test that exposes the bug (if practical)
3. **🟢 Green Phase - Fix Bug**: Implement fix and verify test passes
4. **✅ Validate**: Run full test suite and document the fix

Template response for bug reports:
```
I'll use the red/green approach to fix this bug:

🔴 Red Phase: First, let me reproduce the bug and confirm the issue
🔴 Red Phase: Then implement a test that exposes the problem (if practical)  
🟢 Green Phase: Finally, fix the bug and verify the test passes

Would you like me to proceed with red/green testing for this bug?
```

See `TESTING.md` for detailed guidelines and examples.

### Adding New Commands
1. Follow the `WrapCommandFunc()` pattern for session management
2. Use `GetCobraFlags()` for consistent flag processing
3. Implement CRUD operations using the established query patterns
4. Add enum validation for restricted-value flags

### Working with TrueNAS APIs
- Always use `defaultCallTimeout` (30s) for API calls
- Handle both single results and arrays in API responses
- Use `MaybeBulkApiCall*()` for multiple operations
- Parse job IDs from async operations and wait appropriately

### IPv6 Support
- Use bracket normalization functions in `iscsi_util.go`
- Portal addresses must be properly bracketed for TrueNAS API
- Device path parsing strips brackets for filesystem operations

### Error Patterns
- API errors: Use `core.ExtractApiError*()` functions
- Validation errors: Build error messages with context
- File operations: Wrap with descriptive error messages
- Always provide actionable error messages to users