import redis
import json
import logging
from typing import Any, Optional
from datetime import timedelta

from shared.config import settings

logger = logging.getLogger(__name__)


class RedisClient:
    """Redis client wrapper with JSON serialization support."""

    def __init__(self):
        self.client = redis.Redis(
            host=settings.REDIS_HOST,
            port=settings.REDIS_PORT,
            db=settings.REDIS_DB,
            decode_responses=True
        )

    def get(self, key: str) -> Optional[Any]:
        """Get value from Redis and deserialize from JSON."""
        try:
            value = self.client.get(key)
            if value:
                return json.loads(value)
            return None
        except Exception as e:
            logger.error(f"Redis GET error for key {key}: {e}")
            return None

    def set(self, key: str, value: Any, expire: Optional[int] = None) -> bool:
        """Set value in Redis with JSON serialization."""
        try:
            serialized = json.dumps(value)
            if expire:
                return self.client.setex(key, expire, serialized)
            else:
                return self.client.set(key, serialized)
        except Exception as e:
            logger.error(f"Redis SET error for key {key}: {e}")
            return False

    def delete(self, *keys: str) -> int:
        """Delete one or more keys."""
        try:
            return self.client.delete(*keys)
        except Exception as e:
            logger.error(f"Redis DELETE error: {e}")
            return 0

    def exists(self, key: str) -> bool:
        """Check if key exists."""
        try:
            return self.client.exists(key) > 0
        except Exception as e:
            logger.error(f"Redis EXISTS error for key {key}: {e}")
            return False

    def expire(self, key: str, seconds: int) -> bool:
        """Set expiration time for key."""
        try:
            return self.client.expire(key, seconds)
        except Exception as e:
            logger.error(f"Redis EXPIRE error for key {key}: {e}")
            return False

    def ttl(self, key: str) -> int:
        """Get time to live for key."""
        try:
            return self.client.ttl(key)
        except Exception as e:
            logger.error(f"Redis TTL error for key {key}: {e}")
            return -1

    def incr(self, key: str, amount: int = 1) -> Optional[int]:
        """Increment value of key."""
        try:
            return self.client.incr(key, amount)
        except Exception as e:
            logger.error(f"Redis INCR error for key {key}: {e}")
            return None

    def keys(self, pattern: str = "*"):
        """Get all keys matching pattern."""
        try:
            return self.client.keys(pattern)
        except Exception as e:
            logger.error(f"Redis KEYS error for pattern {pattern}: {e}")
            return []

    def flushdb(self) -> bool:
        """Clear all keys in current database."""
        try:
            return self.client.flushdb()
        except Exception as e:
            logger.error(f"Redis FLUSHDB error: {e}")
            return False

    def ping(self) -> bool:
        """Check Redis connection."""
        try:
            return self.client.ping()
        except Exception as e:
            logger.error(f"Redis PING error: {e}")
            return False


# Global Redis client instance
redis_client = RedisClient()


# Cache key builders
def build_cache_key(prefix: str, *args) -> str:
    """Build cache key from prefix and arguments."""
    return f"{prefix}:" + ":".join(str(arg) for arg in args)


# Common cache operations
def cache_availability(tenant_id: int, master_id: int, date: str, data: dict) -> bool:
    """Cache master availability data."""
    key = build_cache_key("availability", tenant_id, master_id, date)
    return redis_client.set(key, data, expire=300)  # 5 minutes


def get_cached_availability(tenant_id: int, master_id: int, date: str) -> Optional[dict]:
    """Get cached master availability."""
    key = build_cache_key("availability", tenant_id, master_id, date)
    return redis_client.get(key)


def cache_business_info(subdomain: str, data: dict) -> bool:
    """Cache business information."""
    key = build_cache_key("business", subdomain)
    return redis_client.set(key, data, expire=3600)  # 1 hour


def get_cached_business_info(subdomain: str) -> Optional[dict]:
    """Get cached business information."""
    key = build_cache_key("business", subdomain)
    return redis_client.get(key)


def invalidate_cache_pattern(pattern: str) -> int:
    """Invalidate all cache keys matching pattern."""
    keys = redis_client.keys(pattern)
    if keys:
        return redis_client.delete(*keys)
    return 0
