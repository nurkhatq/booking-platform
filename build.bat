@echo off
echo 🚀 Building booking platform...

REM Build base image first
echo 📦 Building base image...
docker build -f Dockerfile.base -t booking-platform-base:latest .

REM Build all services
echo 🔨 Building services...
docker-compose build --parallel

echo ✅ Build complete!
echo 🚀 Starting services...
docker-compose up -d

echo 🎉 Deployment complete!
echo 📊 Check status: docker-compose ps
