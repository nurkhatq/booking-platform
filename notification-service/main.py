from fastapi import FastAPI, HTTPException, status
from pydantic import BaseModel
from typing import Optional
import httpx
import logging
from celery import Celery

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
    title="Notification Service",
    description="Notification service with WhatsApp support",
    version="2.0.0"
)

# Celery app for background tasks
celery_app = Celery(
    "notifications",
    broker=settings.CELERY_BROKER_URL,
    backend=settings.CELERY_RESULT_BACKEND
)

# WhatsApp service URL
WHATSAPP_SERVICE_URL = settings.WHATSAPP_SERVICE_URL


# Request models
class SendWhatsAppRequest(BaseModel):
    phone: str
    message: str


class SendReminderRequest(BaseModel):
    booking_id: int
    phone: str
    message: str


@app.on_event("startup")
async def startup_event():
    """Initialize on startup."""
    logger.info("Starting Notification Service...")
    if check_db_connection():
        logger.info("Database connection successful")


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "notification-service"
    }


@app.post("/send-whatsapp")
async def send_whatsapp(data: SendWhatsAppRequest):
    """
    Send WhatsApp message immediately.
    """
    if not settings.WHATSAPP_ENABLED:
        return {"message": "WhatsApp disabled", "sent": False}

    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{WHATSAPP_SERVICE_URL}/send-message",
                json={
                    "phone": data.phone,
                    "message": data.message
                },
                timeout=10.0
            )

            if response.status_code == 200:
                return {"message": "WhatsApp sent", "sent": True}
            else:
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail="Failed to send WhatsApp message"
                )

    except httpx.RequestError as e:
        logger.error(f"WhatsApp service error: {e}")
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="WhatsApp service unavailable"
        )


@app.post("/schedule-reminder")
async def schedule_reminder(data: SendReminderRequest):
    """
    Schedule a reminder to be sent later via Celery.
    """
    try:
        # Schedule task
        send_reminder_task.apply_async(
            args=[data.phone, data.message],
            countdown=3600  # Send in 1 hour (example)
        )

        return {"message": "Reminder scheduled"}

    except Exception as e:
        logger.error(f"Failed to schedule reminder: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to schedule reminder"
        )


# Celery tasks
@celery_app.task
def send_reminder_task(phone: str, message: str):
    """
    Celery task to send reminder via WhatsApp.
    """
    import requests

    try:
        response = requests.post(
            f"{WHATSAPP_SERVICE_URL}/send-message",
            json={"phone": phone, "message": message},
            timeout=10
        )

        if response.status_code == 200:
            logger.info(f"Reminder sent to {phone}")
        else:
            logger.error(f"Failed to send reminder: {response.text}")

    except Exception as e:
        logger.error(f"Reminder task error: {e}")


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host=settings.NOTIFICATION_SERVICE_HOST if hasattr(settings, 'NOTIFICATION_SERVICE_HOST') else "0.0.0.0",
        port=settings.NOTIFICATION_SERVICE_PORT if hasattr(settings, 'NOTIFICATION_SERVICE_PORT') else 8003,
        reload=settings.DEBUG
    )
