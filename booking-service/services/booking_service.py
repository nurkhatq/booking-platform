from sqlalchemy.orm import Session
from datetime import datetime, date, time, timedelta
from typing import List, Optional
import logging

from shared.models import Booking, Master, MasterSchedule, BookingStatus

logger = logging.getLogger(__name__)


class BookingService:
    """Booking service for business logic."""

    def __init__(self, db: Session):
        self.db = db

    def get_available_slots(
        self,
        master_id: int,
        check_date: date,
        slot_duration: int = 30
    ) -> List[str]:
        """
        Get available time slots for a master on a specific date.

        Args:
            master_id: Master ID
            check_date: Date to check
            slot_duration: Slot duration in minutes (default 30)

        Returns:
            List of available time slots in HH:MM format
        """
        # Get master's schedule for this day of week
        day_of_week = check_date.weekday()
        schedule = self.db.query(MasterSchedule).filter(
            MasterSchedule.master_id == master_id,
            MasterSchedule.day_of_week == day_of_week,
            MasterSchedule.is_working == True
        ).first()

        if not schedule:
            return []

        # Generate all possible slots
        start_time = datetime.combine(check_date, schedule.start_time)
        end_time = datetime.combine(check_date, schedule.end_time)

        all_slots = []
        current_time = start_time

        while current_time + timedelta(minutes=slot_duration) <= end_time:
            all_slots.append(current_time)
            current_time += timedelta(minutes=slot_duration)

        # Get existing bookings for this master on this date
        start_of_day = datetime.combine(check_date, time.min)
        end_of_day = datetime.combine(check_date, time.max)

        bookings = self.db.query(Booking).filter(
            Booking.master_id == master_id,
            Booking.booking_date >= start_of_day,
            Booking.booking_date <= end_of_day,
            Booking.status.in_([BookingStatus.PENDING, BookingStatus.CONFIRMED])
        ).all()

        # Filter out booked slots
        available_slots = []
        for slot in all_slots:
            is_available = True

            for booking in bookings:
                booking_end = booking.booking_date + timedelta(minutes=booking.duration_minutes)

                # Check if slot overlaps with existing booking
                slot_end = slot + timedelta(minutes=slot_duration)

                if (slot < booking_end and slot_end > booking.booking_date):
                    is_available = False
                    break

            if is_available:
                available_slots.append(slot.strftime("%H:%M"))

        return available_slots

    def is_slot_available(
        self,
        master_id: int,
        booking_datetime: datetime,
        duration_minutes: int
    ) -> bool:
        """
        Check if a specific time slot is available.

        Args:
            master_id: Master ID
            booking_datetime: Booking start datetime
            duration_minutes: Booking duration

        Returns:
            True if slot is available, False otherwise
        """
        booking_end = booking_datetime + timedelta(minutes=duration_minutes)

        # Check for overlapping bookings
        overlapping = self.db.query(Booking).filter(
            Booking.master_id == master_id,
            Booking.status.in_([BookingStatus.PENDING, BookingStatus.CONFIRMED]),
            Booking.booking_date < booking_end,
            (Booking.booking_date + timedelta(minutes=Booking.duration_minutes)) > booking_datetime
        ).first()

        if overlapping:
            return False

        # Check if time is within master's working hours
        day_of_week = booking_datetime.weekday()
        schedule = self.db.query(MasterSchedule).filter(
            MasterSchedule.master_id == master_id,
            MasterSchedule.day_of_week == day_of_week,
            MasterSchedule.is_working == True
        ).first()

        if not schedule:
            return False

        booking_time = booking_datetime.time()
        booking_end_time = (booking_datetime + timedelta(minutes=duration_minutes)).time()

        if booking_time < schedule.start_time or booking_end_time > schedule.end_time:
            return False

        return True

    def get_master_bookings(
        self,
        master_id: int,
        start_date: Optional[date] = None,
        end_date: Optional[date] = None
    ) -> List[Booking]:
        """
        Get all bookings for a master within a date range.
        """
        query = self.db.query(Booking).filter(Booking.master_id == master_id)

        if start_date:
            query = query.filter(Booking.booking_date >= datetime.combine(start_date, time.min))

        if end_date:
            query = query.filter(Booking.booking_date <= datetime.combine(end_date, time.max))

        return query.order_by(Booking.booking_date).all()
