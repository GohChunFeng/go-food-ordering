#!/bin/bash
echo "Building the CLI Application..."

# 1. Build the binary
go build -o mcdonalds-bot main.go

# 2. Grant execute permissions explicitly (Fixes your issue)
chmod +x mcdonalds-bot

echo "Build complete."