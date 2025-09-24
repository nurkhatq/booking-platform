# Booking Platform Backend

A comprehensive production-ready multi-tenant booking platform backend built with Go, featuring microservices architecture, gRPC communication, and background job processing.

## Features

- **Multi-tenant Architecture**: Support for multiple businesses with subdomain-based routing
- **Multi-location Support**: Businesses can have multiple branches/locations
- **Role-based Access Control**: Owner, Manager, Master, Client, and Super Admin roles
- **Internationalization**: Support for English, Russian, and Kazakh languages
- **Background Job Processing**: Email notifications, SMS, reminders, and analytics
- **Real-time Availability**: Redis-cached booking availability system
- **Comprehensive API**: RESTful HTTP API with gRPC internal communication
- **Production Ready**: Docker containerization with health checks and monitoring

## Architecture

### Services
- **API Gateway**: HTTP routing, authentication, rate limiting
- **User Service**: Authentication, tenant management, user management
- **Booking Service**: Core booking logic, availability checking
- **Notification Service**: Email/SMS notifications, background job processing
- **Payment Service**: Payment processing and subscription management (stub)
- **Admin Service**: Platform administration and analytics

### Technology Stack
- **Backend**: Go 1.21 with Gin framework
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Communication**: gRPC for internal services, REST for external API
- **Authentication**: JWT with role-based access control
- **Containerization**: Docker & Docker Compose
- **Reverse Proxy**: Nginx with SSL termination

## Quick Start

### Prerequisites
- Docker and Docker Compose
- SSL certificates for your domain
- SMTP credentials for email notifications
- SMS provider credentials (optional)

### 1. Clone and Setup
```bash
git clone 
cd booking-platform
cp .env.example .env
# Edit .env with your configuration
```

### 2. SSL Certificates
Place your SSL certificates at:
- `/etc/ssl/certs/jazyl.tech.pem`
- `/etc/ssl/private/jazyl.tech.key`

Or update the paths in your `.env` file.

### 3. Deploy
```bash
# Make deploy script executable
chmod +x deploy.sh

# Deploy to production
./deploy.sh
```

### 4. Access the Platform
- Main Platform: https://jazyl.tech
- Admin Panel: https://admin.jazyl.tech
- API Documentation: https://jazyl.tech/api/docs

### Default Credentials
- **Super Admin**: admin@jazyl.tech / admin123
- **⚠️ Change default password immediately in production!**

## Development

### Development Setup
```bash
# Install Go 1.21+
# Install Docker and Docker Compose

# Setup development environment
make setup
make dev

# View logs
make logs

# Run tests
make test
```

### Available Make Commands
- `make help` - Show available commands
- `make build` - Build all services
- `make start` - Start all services
- `make stop` - Stop all services
- `make logs` - View logs
- `make test` - Run tests
- `make clean` - Clean up containers
- `make health` - Check service health
- `make backup-db` - Backup database
- `make migrate` - Run database migrations

## API Documentation

### Authentication Endpoints
- `POST /api/v1/register` - Register business
- `POST /api/v1/login` - User login
- `POST /api/v1/logout` - User logout
- `POST /api/v1/refresh-token` - Refresh JWT token

### Public Booking Endpoints
- `GET /api/v1/public/business/:subdomain` - Get business info
- `GET /api/v1/public/business/:subdomain/services` - Get services
- `GET /api/v1/public/business/:subdomain/masters` - Get masters
- `GET /api/v1/public/business/:subdomain/availability` - Check availability
- `POST /api/v1/public/client/verify` - Verify client
- `POST /api/v1/public/booking` - Create booking

### Authenticated Endpoints
All authenticated endpoints require `Authorization: Bearer <token>` header.

#### Booking Management
- `GET /api/v1/bookings` - Get bookings (filtered by role)
- `POST /api/v1/booking` - Create booking
- `PUT /api/v1/booking/:id` - Update booking
- `DELETE /api/v1/booking/:id` - Cancel booking
- `POST /api/v1/booking/:id/complete` - Mark booking complete

#### Service Management (Owner/Manager only)
- `GET /api/v1/services` - Get services
- `POST /api/v1/service` - Create service
- `PUT /api/v1/service/:id` - Update service
- `DELETE /api/v1/service/:id` - Delete service

#### Master Management (Owner/Manager only)
- `GET /api/v1/masters` - Get masters
- `POST /api/v1/master` - Create master
- `PUT /api/v1/master/:id` - Update master
- `DELETE /api/v1/master/:id` - Delete master

#### Admin Endpoints (Super Admin only)
- `GET /api/v1/admin/tenants` - Get pending tenants
- `PUT /api/v1/admin/tenant/:id/approve` - Approve tenant
- `PUT /api/v1/admin/tenant/:id/reject` - Reject tenant
- `GET /api/v1/admin/statistics` - Platform statistics

## Configuration

### Environment Variables
All configuration is done via environment variables. See `.env.example` for all available options.

### Key Configuration Areas
- **Domain Setup**: Configure your domains and SSL certificates
- **Database**: PostgreSQL connection settings
- **Redis**: Cache and session storage
- **Email/SMS**: Notification provider credentials
- **JWT**: Authentication token configuration
- **Business Rules**: Trial periods, cancellation policies, etc.

## Multi-Tenant Architecture

### Domain Structure
- `jazyl.tech` - Main platform for business registration
- `admin.jazyl.tech` - Super admin panel
- `{business}.jazyl.tech` - Public booking site for each business
- `owner.{business}.jazyl.tech` - Business owner dashboard
- `manager.{business}.jazyl.tech` - Location manager dashboard
- `master.{business}.jazyl.tech` - Service provider dashboard

### User Roles
- **SUPER_ADMIN**: Platform administrator
- **OWNER**: Business owner with full access to all locations
- **MANAGER**: Location manager with location-specific access
- **MASTER**: Service provider with booking management
- **CLIENT**: End customers booking appointments

### Multi-Location Support
- One business can have multiple locations/branches
- Location-specific managers and staff
- Unified booking system across all locations
- Location-specific analytics and reporting

## Background Jobs

The platform includes a robust background job system for:

### Job Types
- **Email Notifications**: Booking confirmations, reminders, cancellations
- **SMS Notifications**: Verification codes, urgent reminders
- **Booking Reminders**: Automated 24h and 2h before appointment
- **Analytics Updates**: Daily statistics processing
- **Trial Expiration**: Monitor and notify about trial periods
- **System Health Checks**: Automated system monitoring

### Job Management
- Automatic retry with exponential backoff
- Job scheduling and delayed execution
- Redis-based job queue with persistence
- Configurable worker count and retry policies

## Monitoring and Health Checks

### Health Check Endpoints
- `/health` - Service health check (all services)
- Individual service health checks on their HTTP ports

### System Monitoring
- Database connectivity monitoring
- Redis connectivity monitoring
- gRPC service health checks
- Response time monitoring
- Error rate tracking

### Logging
- Structured JSON logging
- Request/response logging
- Error tracking and reporting
- Admin action audit logs

## Security Features

### Authentication & Authorization
- JWT-based authentication with refresh tokens
- Role-based access control (RBAC)
- Session management for clients
- Secure password hashing with bcrypt

### Security Headers
- CORS configuration
- Security headers (X-Frame-Options, X-Content-Type-Options, etc.)
- HTTPS enforcement
- Rate limiting

### Data Protection
- Input validation and sanitization
- SQL injection prevention
- XSS protection
- Secure cookie configuration

## Caching Strategy

### Redis Caching
- **Booking Availability**: 5-minute cache for frequently accessed data
- **Master Schedules**: 30-minute cache for moderate frequency access
- **Service Lists**: 1-hour cache for relatively static data
- **Business Information**: 1-hour cache for rarely changing data
- **Client Sessions**: 30-day expiration for user convenience
- **Rate Limiting**: 1-minute sliding window counters

### Cache Invalidation
- Automatic cache invalidation on data changes
- Event-driven cache updates
- Manual cache clearing capabilities

## Internationalization (i18n)

### Supported Languages
- English (en) - Default
- Russian (ru)
- Kazakh (kk)

### Translation System
- Key-based translation system
- JSON translation files
- Dynamic language detection from headers
- Localized email templates
- Error message localization

## Database Schema

### Core Tables
- `tenants` - Business information and settings
- `locations` - Business locations/branches
- `users` - All platform users with role-based access
- `masters` - Service providers with specializations
- `services` - Available services with pricing
- `bookings` - Appointment bookings with full lifecycle
- `client_sessions` - Simplified client authentication
- `permission_requests` - Master permission management

### Relationships
- Multi-tenant isolation
- Location-based organization
- Role-based access control
- Audit trail for all actions

## Deployment

### Production Deployment
1. Setup SSL certificates
2. Configure environment variables
3. Run deployment script: `./deploy.sh`
4. Monitor service health

### Docker Services
- PostgreSQL with persistent volume
- Redis with persistent volume
- Nginx reverse proxy with SSL
- 6 microservices with health checks
- Automatic service discovery and networking

### Scaling Considerations
- Horizontal scaling of microservices
- Database connection pooling
- Redis clustering for high availability
- Load balancing across service instances

## API Rate Limiting

- 100 requests per minute per IP (configurable)
- Different limits for different user roles
- Sliding window rate limiting
- Redis-based distributed rate limiting

## Error Handling

### Error Response Format
```json
{
  "error": "error_code",
  "message": "Human readable error message",
  "details": "Additional error details if available"
}
```

### HTTP Status Codes
- 200: Success
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 403: Forbidden
- 404: Not Found
- 429: Rate Limited
- 500: Internal Server Error

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request

### Code Style
- Follow Go conventions
- Use gofmt for formatting
- Write comprehensive tests
- Document public APIs

## License

This project is proprietary software. All rights reserved.

## Support

For support and questions:
- Create an issue in the repository
- Contact: support@jazyl.tech
- Documentation: https://docs.jazyl.tech