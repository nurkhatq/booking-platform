#!/bin/bash
# build.sh
set -e

echo "🚀 Building booking platform..."

# Build base image first
echo "📦 Building base image..."
docker build -f Dockerfile.base -t booking-platform-base:latest .

# Build all services
echo "🔨 Building services..."
docker-compose build --parallel

echo "✅ Build complete!"
echo "🚀 Starting services..."
docker-compose up -d

echo "🎉 Deployment complete!"
echo "📊 Check status: docker-compose ps"
