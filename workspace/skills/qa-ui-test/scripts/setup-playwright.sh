#!/usr/bin/env bash
set -euo pipefail

# Install Playwright and its browser dependencies
echo "Installing Playwright..."
npm init -y 2>/dev/null || true
npm install playwright 2>/dev/null

echo "Installing Chromium browser..."
npx playwright install chromium

echo "Installing system dependencies..."
npx playwright install-deps chromium 2>/dev/null || echo "Warning: Could not install system deps. Run with sudo if needed."

echo "Playwright setup complete."
