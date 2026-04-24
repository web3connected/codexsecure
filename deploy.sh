#!/bin/bash
# CodexSecure — Deploy Entry Point
# Delegates to scripts/deployment/forge-deploy.sh (same pattern as CodexHash)
#
# Usage (on server or in Forge deploy script field):
#   bash deploy.sh

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec bash "$SCRIPT_DIR/scripts/deployment/forge-deploy.sh" "$@"
