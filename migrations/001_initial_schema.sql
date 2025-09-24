-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE user_role AS ENUM ('SUPER_ADMIN', 'OWNER', 'MANAGER', 'MASTER', 'CLIENT');
CREATE TYPE business_type AS ENUM ('barbershop', 'salon', 'clinic', 'spa', 'beauty_center');
CREATE TYPE booking_status AS ENUM ('pending', 'confirmed', 'completed', 'cancelled');
CREATE TYPE tenant_status AS ENUM ('pending', 'approved', 'active', 'suspended', 'expired');
CREATE TYPE permission_status AS ENUM ('pending', 'approved', 'denied');

-- Tenants (Businesses)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_name VARCHAR(255) NOT NULL,
    business_type business_type NOT NULL,
    subdomain VARCHAR(100) UNIQUE NOT NULL,
    timezone VARCHAR(100) NOT NULL DEFAULT 'Asia/Almaty',
    owner_id UUID,
    status tenant_status DEFAULT 'pending',
    trial_start_date TIMESTAMP,
    trial_end_date TIMESTAMP,
    subscription_end_date TIMESTAMP,
    settings JSONB DEFAULT '{}',
    branding JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Locations (Branches)
CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    manager_id UUID,
    working_hours JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    location_id UUID REFERENCES locations(id) ON DELETE SET NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role user_role NOT NULL,
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    phone_verified BOOLEAN DEFAULT false,
    last_login TIMESTAMP,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Masters (Service Providers)
CREATE TABLE masters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE,
    bio TEXT,
    photo_url VARCHAR(500),
    specialization TEXT,
    experience_years INTEGER DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.0,
    total_reviews INTEGER DEFAULT 0,
    permissions JSONB DEFAULT '{}',
    availability JSONB DEFAULT '{}',
    is_visible BOOLEAN DEFAULT true,
    is_accepting_bookings BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Services
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    base_price DECIMAL(10,2) NOT NULL,
    base_duration INTEGER NOT NULL, -- minutes
    is_active BOOLEAN DEFAULT true,
    popularity_score INTEGER DEFAULT 0,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Location Services (Service availability per location)
CREATE TABLE location_services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE,
    service_id UUID REFERENCES services(id) ON DELETE CASCADE,
    price DECIMAL(10,2),
    duration INTEGER, -- minutes
    is_available BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(location_id, service_id)
);

-- Master Services (Individual master pricing)
CREATE TABLE master_services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID REFERENCES masters(id) ON DELETE CASCADE,
    service_id UUID REFERENCES services(id) ON DELETE CASCADE,
    price DECIMAL(10,2),
    duration INTEGER, -- minutes
    is_available BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(master_id, service_id)
);

-- Client Sessions (Simplified authentication)
CREATE TABLE client_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    verification_code VARCHAR(10),
    verification_expires TIMESTAMP,
    is_verified BOOLEAN DEFAULT false,
    session_token VARCHAR(500),
    session_expires TIMESTAMP,
    last_used TIMESTAMP,
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Bookings
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE,
    master_id UUID REFERENCES masters(id) ON DELETE CASCADE,
    service_id UUID REFERENCES services(id) ON DELETE CASCADE,
    client_session_id UUID REFERENCES client_sessions(id) ON DELETE CASCADE,
    
    -- Client Information
    client_name VARCHAR(255) NOT NULL,
    client_email VARCHAR(255) NOT NULL,
    client_phone VARCHAR(20) NOT NULL,
    
    -- Booking Details
    booking_date DATE NOT NULL,
    booking_time TIME NOT NULL,
    duration INTEGER NOT NULL, -- minutes
    price DECIMAL(10,2) NOT NULL,
    status booking_status DEFAULT 'confirmed',
    
    -- Notes and Additional Info
    client_notes TEXT,
    master_notes TEXT,
    cancellation_reason TEXT,
    cancelled_by VARCHAR(100),
    cancelled_at TIMESTAMP,
    
    -- Booking Management
    created_by VARCHAR(100) DEFAULT 'client',
    confirmation_code VARCHAR(20),
    reminder_sent_24h BOOLEAN DEFAULT false,
    reminder_sent_2h BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Permission Requests
CREATE TABLE permission_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID REFERENCES masters(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    permission_type VARCHAR(100) NOT NULL,
    current_value JSONB,
    requested_value JSONB,
    reason TEXT,
    status permission_status DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP,
    response_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_tenants_subdomain ON tenants(subdomain);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant_role ON users(tenant_id, role);
CREATE INDEX idx_masters_tenant_location ON masters(tenant_id, location_id);
CREATE INDEX idx_bookings_date_master ON bookings(booking_date, master_id);
CREATE INDEX idx_bookings_tenant_status ON bookings(tenant_id, status);
CREATE INDEX idx_bookings_client_email ON bookings(client_email);
CREATE INDEX idx_client_sessions_email ON client_sessions(email);
CREATE INDEX idx_client_sessions_token ON client_sessions(session_token);

-- Add foreign key constraints
ALTER TABLE tenants ADD CONSTRAINT fk_tenants_owner FOREIGN KEY (owner_id) REFERENCES users(id);
ALTER TABLE locations ADD CONSTRAINT fk_locations_manager FOREIGN KEY (manager_id) REFERENCES users(id);

-- Insert initial super admin user
INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, email_verified)
VALUES (
    uuid_generate_v4(),
    'admin@jazyl.tech',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewMNaZ8VgXFGZ9NS', -- password: admin123
    'Super',
    'Admin',
    'SUPER_ADMIN',
    true,
    true
);