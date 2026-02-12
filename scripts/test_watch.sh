#!/bin/bash
# Watch mode for running unit tests on file changes
# Requires: fswatch (brew install fswatch on macOS)

echo "🔍 Watching for file changes..."
echo "Running unit tests on changes to .go files"
echo "Press Ctrl+C to stop"
echo ""

# Initial test run
make test-unit

# Watch for changes
fswatch -o internal/app/product/domain/*.go | while read change; do
    clear
    echo "📝 Files changed, re-running tests..."
    echo ""
    make test-unit
done
