#!/bin/bash

echo "🚀 Starting CodexHash Static Deployment..."

# Exit on any error
set -e

# Build Next.js application locally
echo "📦 Building Next.js application..."
cd /home/web3codex/projects/codex_hash

# Set production environment variables for build
export NODE_ENV=production
export NEXT_PUBLIC_API_URL=https://codexhash.io/api
export NEXT_PUBLIC_SITE_URL=https://codexhash.io

# Clean and build
rm -rf .next out
npm run build

# Check if we need to export static files
if [ ! -d "out" ]; then
    echo "📤 Exporting static files..."
    npx next export
fi

# Create deployment package
echo "📦 Creating deployment package..."
cd out
tar -czf ../codexhash-static.tar.gz .
cd ..

# Deploy to production server
echo "🌐 Deploying to production server..."
scp codexhash-static.tar.gz forge@45.79.180.207:/tmp/

# Extract on production server and update files
ssh forge@45.79.180.207 << 'EOF'
    cd /home/forge/codexhash.io
    
    # Backup current index.html
    cp index.html index.html.backup.$(date +%Y%m%d_%H%M%S)
    
    # Extract new static files
    cd /tmp
    tar -xzf codexhash-static.tar.gz
    
    # Copy the main index.html to the site root
    cp index.html /home/forge/codexhash.io/
    
    # Copy any other static assets
    if [ -d "_next" ]; then
        cp -r _next /home/forge/codexhash.io/
    fi
    
    # Clean up
    rm -rf /tmp/codexhash-static.tar.gz /tmp/_next /tmp/index.html
    
    echo "✅ Static files deployed successfully"
EOF

# Clean up local files
rm -f codexhash-static.tar.gz

echo "🎉 Deployment completed successfully!"
echo "🌐 Site is available at: https://codexhash.io"