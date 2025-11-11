from sqlalchemy import Column, Integer, String, DateTime, Boolean, ForeignKey, Text, Numeric, Enum as SQLEnum, Time
from sqlalchemy.orm import relationship
from datetime import datetime
from enum import Enum

from shared.database import Base


class UserRole(str, Enum):
    """User roles enum."""
    SUPER_ADMIN = "SUPER_ADMIN"
    OWNER = "OWNER"
    MANAGER = "MANAGER"
    MASTER = "MASTER"
    CLIENT = "CLIENT"


class TenantStatus(str, Enum):
    """Tenant status enum."""
    PENDING = "PENDING"
    ACTIVE = "ACTIVE"
    SUSPENDED = "SUSPENDED"
    TRIAL = "TRIAL"
    REJECTED = "REJECTED"


class BookingStatus(str, Enum):
    """Booking status enum."""
    PENDING = "PENDING"
    CONFIRMED = "CONFIRMED"
    CANCELLED = "CANCELLED"
    COMPLETED = "COMPLETED"
    NO_SHOW = "NO_SHOW"


class Tenant(Base):
    """Business tenant model."""
    __tablename__ = "tenants"

    id = Column(Integer, primary_key=True, index=True)
    subdomain = Column(String(50), unique=True, nullable=False, index=True)
    business_name = Column(String(200), nullable=False)
    phone = Column(String(20), nullable=False)
    email = Column(String(100), nullable=True)
    description = Column(Text, nullable=True)
    status = Column(SQLEnum(TenantStatus), default=TenantStatus.PENDING, nullable=False)
    trial_end_date = Column(DateTime, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    locations = relationship("Location", back_populates="tenant", cascade="all, delete-orphan")
    users = relationship("User", back_populates="tenant")
    services = relationship("Service", back_populates="tenant", cascade="all, delete-orphan")
    masters = relationship("Master", back_populates="tenant", cascade="all, delete-orphan")


class Location(Base):
    """Business location model."""
    __tablename__ = "locations"

    id = Column(Integer, primary_key=True, index=True)
    tenant_id = Column(Integer, ForeignKey("tenants.id", ondelete="CASCADE"), nullable=False)
    name = Column(String(200), nullable=False)
    address = Column(String(500), nullable=True)
    phone = Column(String(20), nullable=True)
    is_main = Column(Boolean, default=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    tenant = relationship("Tenant", back_populates="locations")
    masters = relationship("Master", back_populates="location")


class User(Base):
    """User model."""
    __tablename__ = "users"

    id = Column(Integer, primary_key=True, index=True)
    tenant_id = Column(Integer, ForeignKey("tenants.id"), nullable=True)
    email = Column(String(100), unique=True, nullable=False, index=True)
    phone = Column(String(20), nullable=True, index=True)
    password_hash = Column(String(255), nullable=False)
    full_name = Column(String(200), nullable=False)
    role = Column(SQLEnum(UserRole), nullable=False)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    tenant = relationship("Tenant", back_populates="users")


class Service(Base):
    """Service model."""
    __tablename__ = "services"

    id = Column(Integer, primary_key=True, index=True)
    tenant_id = Column(Integer, ForeignKey("tenants.id", ondelete="CASCADE"), nullable=False)
    name = Column(String(200), nullable=False)
    description = Column(Text, nullable=True)
    duration_minutes = Column(Integer, nullable=False)
    price = Column(Numeric(10, 2), nullable=False)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    tenant = relationship("Tenant", back_populates="services")
    master_services = relationship("MasterService", back_populates="service", cascade="all, delete-orphan")


class Master(Base):
    """Master (service provider) model."""
    __tablename__ = "masters"

    id = Column(Integer, primary_key=True, index=True)
    tenant_id = Column(Integer, ForeignKey("tenants.id", ondelete="CASCADE"), nullable=False)
    location_id = Column(Integer, ForeignKey("locations.id"), nullable=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=True)
    full_name = Column(String(200), nullable=False)
    phone = Column(String(20), nullable=False)
    description = Column(Text, nullable=True)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    tenant = relationship("Tenant", back_populates="masters")
    location = relationship("Location", back_populates="masters")
    master_services = relationship("MasterService", back_populates="master", cascade="all, delete-orphan")
    schedules = relationship("MasterSchedule", back_populates="master", cascade="all, delete-orphan")
    bookings = relationship("Booking", back_populates="master")


class MasterService(Base):
    """Master-Service relationship."""
    __tablename__ = "master_services"

    id = Column(Integer, primary_key=True, index=True)
    master_id = Column(Integer, ForeignKey("masters.id", ondelete="CASCADE"), nullable=False)
    service_id = Column(Integer, ForeignKey("services.id", ondelete="CASCADE"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    # Relationships
    master = relationship("Master", back_populates="master_services")
    service = relationship("Service", back_populates="master_services")


class MasterSchedule(Base):
    """Master working schedule."""
    __tablename__ = "master_schedules"

    id = Column(Integer, primary_key=True, index=True)
    master_id = Column(Integer, ForeignKey("masters.id", ondelete="CASCADE"), nullable=False)
    day_of_week = Column(Integer, nullable=False)  # 0=Monday, 6=Sunday
    start_time = Column(Time, nullable=False)
    end_time = Column(Time, nullable=False)
    is_working = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    master = relationship("Master", back_populates="schedules")


class Client(Base):
    """Client model."""
    __tablename__ = "clients"

    id = Column(Integer, primary_key=True, index=True)
    phone = Column(String(20), unique=True, nullable=False, index=True)
    full_name = Column(String(200), nullable=True)
    email = Column(String(100), nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    bookings = relationship("Booking", back_populates="client")


class Booking(Base):
    """Booking model."""
    __tablename__ = "bookings"

    id = Column(Integer, primary_key=True, index=True)
    tenant_id = Column(Integer, ForeignKey("tenants.id"), nullable=False)
    client_id = Column(Integer, ForeignKey("clients.id"), nullable=False)
    master_id = Column(Integer, ForeignKey("masters.id"), nullable=False)
    service_id = Column(Integer, ForeignKey("services.id"), nullable=False)
    booking_date = Column(DateTime, nullable=False, index=True)
    duration_minutes = Column(Integer, nullable=False)
    price = Column(Numeric(10, 2), nullable=False)
    status = Column(SQLEnum(BookingStatus), default=BookingStatus.PENDING, nullable=False)
    client_notes = Column(Text, nullable=True)
    admin_notes = Column(Text, nullable=True)
    whatsapp_reminder_sent = Column(Boolean, default=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    client = relationship("Client", back_populates="bookings")
    master = relationship("Master", back_populates="bookings")
