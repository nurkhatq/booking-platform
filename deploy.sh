#!/bin/bash

set -e

echo "🚀 Starting Booking Platform Deployment..."

# Check if SSL certificates exist
if [ ! -f "ssl/jazyl.tech.pem" ] || [ ! -f "ssl/jazyl.tech.key" ]; then
    echo "❌ SSL certificates not found!"
    echo "Please ensure SSL certificates are placed at:"
    echo "  - ssl/jazyl.tech.pem"
    echo "  - ssl/jazyl.tech.key"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ .env file not found!"
    echo "Please create .env file with all required environment variables."
    exit 1
fi

# Create necessary directories
echo "📁 Creating directories..."
mkdir -p postgres_data redis_data logs

# Generate dhparam for SSL if it doesn't exist
if [ ! -f "ssl/dhparam.pem" ]; then
    echo "🔐 Generating dhparam for SSL..."
    openssl dhparam -out ssl/dhparam.pem 2048
fi

# Stop existing containers
echo "🛑 Stopping existing containers..."
docker-compose down || true

# Build and start services
echo "🔨 Building and starting services..."
docker-compose up -d --build

# Wait for services to be healthy
echo "⏳ Waiting for services to be healthy..."
sleep 30

# Check service health
echo "🏥 Checking service health..."
services=("postgres" "redis" "api-gateway" "user-service" "booking-service" "notification-service" "payment-service" "admin-service")

for service in "${services[@]}"; do
    if docker-compose ps $service | grep -q "healthy\|Up"; then
        echo "✅ $service is healthy"
    else
        echo "❌ $service is not healthy"
        docker-compose logs $service
    fi
done

# Check if API Gateway is responding
echo "🌐 Testing API Gateway..."
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ API Gateway is responding"
else
    echo "❌ API Gateway is not responding"
    docker-compose logs api-gateway
fi

# Check if SSL is working (if nginx is accessible)
if curl -f -k https://jazyl.tech/health > /dev/null 2>&1; then
    echo "✅ HTTPS is working"
else
    echo "❌ HTTPS is not working - check nginx configuration"
    docker-compose logs nginx
fi

echo ""
echo "🎉 Deployment completed!"
echo ""
echo "Services are running at:"
echo "  - Main Platform: https://jazyl.tech"
echo "  - Admin Panel: https://admin.jazyl.tech"
echo "  - API Gateway: http://localhost:8080"
echo ""
echo "To check logs: docker-compose logs [service-name]"
echo "To stop services: docker-compose down"
echo "To restart services: docker-compose restart [service-name]"
echo ""
echo "Default super admin credentials:"
echo "  Email: admin@jazyl.tech"
echo "  Password: admin123"
echo ""
echo "⚠️  Remember to change the default admin password in production!"