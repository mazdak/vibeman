#!/bin/bash
# Integration test runner for Vibeman

set -e

echo "Running Vibeman Integration Tests"
echo "================================"

# Set up environment
export VIBEMAN_TEST_MODE=integration

# Clean up any previous test artifacts
rm -rf /tmp/vibeman-integration-*

# Run all integration tests with build tag
echo "Running integration tests..."
go test -v -tags=integration ./internal/integration/...

# Check if tests passed
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ All integration tests passed!"
else
    echo ""
    echo "❌ Some integration tests failed"
    exit 1
fi