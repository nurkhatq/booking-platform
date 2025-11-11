from fastapi import FastAPI
import logging

from shared.config import settings
from shared.database import check_db_connection

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="Payment Service",
    description="Payment processing service (stub)",
    version="2.0.0"
)


@app.on_event("startup")
async def startup_event():
    """Initialize on startup."""
    logger.info("Starting Payment Service...")


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "payment-service"
    }


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "message": "Payment Service - Coming Soon",
        "version": "2.0.0"
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host=settings.PAYMENT_SERVICE_HOST if hasattr(settings, 'PAYMENT_SERVICE_HOST') else "0.0.0.0",
        port=settings.PAYMENT_SERVICE_PORT if hasattr(settings, 'PAYMENT_SERVICE_PORT') else 8004,
        reload=settings.DEBUG
    )
