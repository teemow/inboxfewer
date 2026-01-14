#!/bin/bash
# get-latest-version.sh
#
# Gets the latest release version from git tags.
# Outputs the version without the 'v' prefix.
#
# Usage:
#   ./scripts/get-latest-version.sh
#
# Output:
#   Prints the latest version (e.g., "0.0.30") to stdout.
#   Falls back to "0.0.0" if no tags exist.

set -euo pipefail

# Fetch all tags from origin to ensure we have the latest
git fetch origin --tags --force >/dev/null 2>&1 || true

# Get the latest tag sorted by version
LATEST_TAG=$(git tag --sort=-version:refname | head -n1 || true)

# Handle empty result (no tags exist)
if [ -z "$LATEST_TAG" ]; then
  LATEST_TAG="v0.0.0"
fi

# Strip 'v' prefix and output
echo "${LATEST_TAG#v}"
