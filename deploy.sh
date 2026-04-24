#!/bin/bash
# CodexSecure — Forge Server-Side Deploy Script
#
# PASTE THIS INTO: Laravel Forge > Sites > codexsecure.io > Deploy Script
# This runs ON the server, no external SSH needed.
#
# Also works as a local deploy if SSH is available:
#   ssh forge@45.79.180.207 'bash -s' < deploy.sh

set -e

APP_DIR="/home/forge/codexsecure.io"
APP_NAME="codexsecure-prod"
PORT=3003

echo "🚀 CodexSecure deploy starting..."
cd "$APP_DIR"

echo "📥 Pulling latest code..."
git pull origin main

echo "📦 Installing dependencies..."
npm ci --production=false

echo "🏗️  Building Next.js..."
NEXT_PUBLIC_API_URL=https://codexsecure.io/api \
NEXT_PUBLIC_SITE_URL=https://codexsecure.io \
NODE_ENV=production \
npm run build

mkdir -p logs

echo "🔄 Restarting PM2..."
pm2 stop codexsecure-dev  2>/dev/null || true
pm2 delete codexsecure-dev 2>/dev/null || true
pm2 stop codexsecure      2>/dev/null || true
pm2 delete codexsecure     2>/dev/null || true
pm2 stop "$APP_NAME"      2>/dev/null || true
pm2 delete "$APP_NAME"     2>/dev/null || true
pm2 start ecosystem.config.js --only "$APP_NAME"
pm2 save

echo "⏳ Waiting for app..."
sleep 8

if curl -sf "http://localhost:$PORT" > /dev/null; then
  echo "✅ $APP_NAME is live on port $PORT"
else
  echo "❌ Health check failed — logs:"
  pm2 logs "$APP_NAME" --lines 30 --nostream
  exit 1
fi

echo "🎉 Deploy complete → https://codexsecure.io"
