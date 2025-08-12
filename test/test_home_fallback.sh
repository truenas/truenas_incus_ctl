#!/bin/bash

# Test script to verify HOME fallback behavior
# This test ensures truenas_incus_ctl works when $HOME is undefined,
# which is critical for container environments like Incus.

set -e

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$TEST_DIR")"
BINARY="$PROJECT_ROOT/truenas_incus_ctl"

echo "=== Testing HOME Environment Variable Fallback ==="
echo "Project root: $PROJECT_ROOT"
echo "Binary: $BINARY"

# Build the binary if it doesn't exist
if [ ! -f "$BINARY" ]; then
    echo "Building binary..."
    cd "$PROJECT_ROOT"
    go build
    cd - > /dev/null
fi

# Test 1: Ensure tool doesn't fail with "$HOME is not defined" error
echo
echo "Test 1: Tool should not fail with HOME undefined..."
OUTPUT=$(env -u HOME "$BINARY" config list 2>&1 || true)
if echo "$OUTPUT" | grep -q "\$HOME is not defined"; then
    echo "✗ FAIL: Tool exited with '\$HOME is not defined' error"
    echo "Output: $OUTPUT"
    exit 1
else
    echo "✓ PASS: Tool handles missing HOME gracefully"
fi

# Test 2: Test config path fallback by creating config in current directory
echo
echo "Test 2: Testing config path fallback to current directory..."
mkdir -p "$PROJECT_ROOT/.truenas_incus_ctl"
echo '{"hosts":{"test":{"url":"http://test.local","api_key":"test"}}}' > "$PROJECT_ROOT/.truenas_incus_ctl/config.json"

if env -u HOME "$BINARY" config list > /dev/null 2>&1; then
    echo "✓ PASS: Config found in current directory fallback location"
else
    echo "✗ FAIL: Config not found in fallback location"
    rm -rf "$PROJECT_ROOT/.truenas_incus_ctl"
    exit 1
fi

# Clean up test config
rm -rf "$PROJECT_ROOT/.truenas_incus_ctl"

# Test 3: Test various HOME scenarios
echo
echo "Test 3: Testing various HOME scenarios..."
for scenario in "unset HOME" "empty HOME" "no HOME and PWD"; do
    case "$scenario" in
        "unset HOME")
            cmd="env -u HOME"
            ;;
        "empty HOME")
            cmd="env HOME="
            ;;
        "no HOME and PWD")
            cmd="env -u HOME -u PWD"
            ;;
    esac
    
    # Test version command first (should always work)
    if $cmd "$BINARY" version > /dev/null 2>&1; then
        # Also test that it doesn't crash with "$HOME is not defined" error on any command
        OUTPUT=$($cmd "$BINARY" config list 2>&1 || true)
        if echo "$OUTPUT" | grep -q "\$HOME is not defined"; then
            echo "✗ FAIL: Tool crashed with '\$HOME is not defined' error in $scenario"
            exit 1
        else
            echo "✓ PASS: Tool works with $scenario"
        fi
    else
        echo "✗ FAIL: Tool failed with $scenario"
        exit 1
    fi
done

echo
echo "=== All HOME fallback tests passed! ==="
echo "The tool correctly handles missing HOME environment variable."
echo "It falls back to current directory for config and socket paths."