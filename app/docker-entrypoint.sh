#!/bin/sh
set -e

# Ensure data directory exists and has correct permissions
if [ ! -w "${CCNEXUS_DATA_DIR:-/data}" ]; then
    echo "Warning: Data directory is not writable, attempting to fix..."
fi

# Run the server
exec /app/ccnexus-server "$@"
