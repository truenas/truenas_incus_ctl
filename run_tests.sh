#!/bin/bash

# Run all tests for truenas_incus_ctl
set -e

echo "=== Running Go unit tests ==="
go test -v ./cmd
go test -v ./core

echo
echo "=== Running integration tests ==="
./test/test_home_fallback.sh

echo
echo "=== All tests passed! ==="