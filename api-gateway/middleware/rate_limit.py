from fastapi import Request, HTTPException, status
from fastapi.responses import JSONResponse
import time
import logging

from shared.cache import redis_client
from shared.config import settings

logger = logging.getLogger(__name__)


async def rate_limit_middleware(request: Request, call_next):
    """
    Rate limiting middleware using Redis.

    Implements sliding window rate limiting per IP address.
    """
    # Skip rate limiting for health checks
    if request.url.path in ["/health", "/api/docs", "/api/redoc", "/openapi.json"]:
        return await call_next(request)

    # Get client IP
    client_ip = request.client.host
    if not client_ip:
        return await call_next(request)

    # Build rate limit key
    key = f"rate_limit:{client_ip}"
    current_minute = int(time.time() / 60)
    rate_limit_key = f"{key}:{current_minute}"

    try:
        # Get current request count
        current_count = redis_client.get(rate_limit_key)

        if current_count is None:
            # First request in this minute
            redis_client.set(rate_limit_key, 1, expire=60)
        else:
            # Increment counter
            if int(current_count) >= settings.RATE_LIMIT_PER_MINUTE:
                return JSONResponse(
                    status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                    content={
                        "error": "Rate limit exceeded",
                        "detail": f"Maximum {settings.RATE_LIMIT_PER_MINUTE} requests per minute"
                    }
                )

            redis_client.incr(rate_limit_key)

    except Exception as e:
        logger.error(f"Rate limit check failed: {e}")
        # If Redis fails, allow request to proceed
        pass

    return await call_next(request)
