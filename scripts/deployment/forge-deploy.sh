#!/bin/bash

# CodexSecure.io Forge Deployment Script
# Paste into: Laravel Forge > Sites > codexsecure.io > Deploy Script
# Or run manually on server: bash /home/forge/codexsecure.io/scripts/deployment/forge-deploy.sh

set -e

echo "🚀 Starting CodexSecure.io deployment..."

cd $FORGE_SITE_PATH

# Pull latest changes
git pull origin $FORGE_SITE_BRANCH

# Install Node.js dependencies
echo "📦 Installing dependencies..."
npm ci --production=false

# Build Next.js application
echo "🔨 Building Next.js..."
NEXT_PUBLIC_API_URL=https://codexsecure.io/api \
NEXT_PUBLIC_SITE_URL=https://codexsecure.io \
NODE_ENV=production \
npm run build

# Create logs directory if it doesn't exist
mkdir -p logs

# Install PM2 if not already installed
if ! command -v pm2 &> /dev/null; then
    echo "📦 Installing PM2..."
    npm install -g pm2
fi

# Stop and clean all codexsecure processes
echo "🛑 Stopping existing processes..."
pm2 stop codexsecure-dev  2>/dev/null || true
pm2 delete codexsecure-dev 2>/dev/null || true
pm2 stop codexsecure      2>/dev/null || true
pm2 delete codexsecure     2>/dev/null || true
pm2 stop codexsecure-prod 2>/dev/null || true
pm2 delete codexsecure-prod 2>/dev/null || true

# Start production process
echo "▶️  Starting codexsecure-prod on port 3003..."
pm2 start ecosystem.config.js --only codexsecure-prod
pm2 save

echo "⏳ Waiting for app to start..."
sleep 8

# Health check
echo "🔍 Health check..."
if curl -sf http://localhost:3003 > /dev/null; then
    echo "✅ CodexSecure is live on port 3003"
else
    echo "❌ Health check failed — PM2 logs:"
    pm2 logs codexsecure-prod --lines 30 --nostream
    exit 1
fi

echo "🎉 CodexSecure.io deployment complete!"
