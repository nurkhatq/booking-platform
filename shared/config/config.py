from pydantic_settings import BaseSettings
from typing import List


class Settings(BaseSettings):
    # Environment
    ENVIRONMENT: str = "development"
    DEBUG: bool = True
    LOG_LEVEL: str = "INFO"

    # Domain
    BASE_DOMAIN: str = "jazyl.tech"
    MAIN_DOMAIN: str = "jazyl.tech"
    ADMIN_DOMAIN: str = "admin.jazyl.tech"

    # Database
    DATABASE_URL: str
    POSTGRES_USER: str = "booking_user"
    POSTGRES_PASSWORD: str = "booking_password"
    POSTGRES_DB: str = "booking_platform"
    DB_POOL_SIZE: int = 20
    DB_MAX_OVERFLOW: int = 10

    # Redis
    REDIS_URL: str = "redis://redis:6379/0"
    REDIS_HOST: str = "redis"
    REDIS_PORT: int = 6379
    REDIS_DB: int = 0

    # JWT
    JWT_SECRET_KEY: str
    JWT_ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 1440
    REFRESH_TOKEN_EXPIRE_DAYS: int = 7

    # WhatsApp
    WHATSAPP_SERVICE_URL: str = "http://whatsapp-service:3000"
    WHATSAPP_ENABLED: bool = True

    # Security
    BCRYPT_ROUNDS: int = 12
    RATE_LIMIT_PER_MINUTE: int = 100
    CORS_ORIGINS: str = "*"

    # Business Logic
    DEFAULT_TRIAL_DAYS: int = 30
    BOOKING_ADVANCE_LIMIT_DAYS: int = 30
    CANCELLATION_HOURS: int = 2
    REMINDER_HOURS: str = "24,2"

    # i18n
    DEFAULT_LANGUAGE: str = "ru"
    SUPPORTED_LANGUAGES: str = "ru,en,kk"

    # Celery
    CELERY_BROKER_URL: str = "redis://redis:6379/1"
    CELERY_RESULT_BACKEND: str = "redis://redis:6379/2"

    class Config:
        env_file = ".env"
        case_sensitive = True

    @property
    def cors_origins_list(self) -> List[str]:
        if self.CORS_ORIGINS == "*":
            return ["*"]
        return [origin.strip() for origin in self.CORS_ORIGINS.split(",")]

    @property
    def supported_languages_list(self) -> List[str]:
        return [lang.strip() for lang in self.SUPPORTED_LANGUAGES.split(",")]

    @property
    def reminder_hours_list(self) -> List[int]:
        return [int(h.strip()) for h in self.REMINDER_HOURS.split(",")]


settings = Settings()
