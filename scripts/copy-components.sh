#!/bin/bash
# Copy web3codex-components from SDK to node_modules
# This avoids symlink issues with Next.js/Turbopack

SOURCE_DIR="/home/web3codex/projects/SDK/web3codex-components"
TARGET_DIR="./node_modules/@web3codex/components"

echo "📦 Copying web3codex-components..."

# Remove existing symlink or directory
rm -rf "$TARGET_DIR"

# Create parent directory if it doesn't exist
mkdir -p "$(dirname "$TARGET_DIR")"

# Copy the entire package
cp -r "$SOURCE_DIR" "$TARGET_DIR"

echo "✅ Components copied successfully to $TARGET_DIR"
