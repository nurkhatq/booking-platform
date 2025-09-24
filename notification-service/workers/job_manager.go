package workers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    
    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
    
    "booking-platform/shared/config"
    "booking-platform/shared/cache"
    "booking-platform/shared/database"
)

type JobManager struct {
    config   *config.Config
    redis    *redis.Client
    workers  []*Worker
    stopChan chan struct{}
}

type JobType string

const (
    JobEmailNotification     JobType = "email_notification"
    JobSMSNotification      JobType = "sms_notification"
    JobBookingReminder      JobType = "booking_reminder"
    JobAnalyticsUpdate      JobType = "analytics_update"
    JobTrialExpiration      JobType = "trial_expiration"
    JobFailedPayment        JobType = "failed_payment"
    JobSystemHealthCheck    JobType = "system_health_check"
)

type Job struct {
    ID            string                 `json:"id"`
    Type          JobType               `json:"type"`
    Data          map[string]interface{} `json:"data"`
    ScheduledTime time.Time             `json:"scheduled_time"`
    CreatedAt     time.Time             `json:"created_at"`
    Attempts      int                   `json:"attempts"`
    MaxAttempts   int                   `json:"max_attempts"`
    Status        string                `json:"status"` // pending, processing, completed, failed
    Error         string                `json:"error,omitempty"`
}

type Worker struct {
    id       string
    manager  *JobManager
    stopChan chan struct{}
}

func NewJobManager(cfg *config.Config) *JobManager {
    return &JobManager{
        config:   cfg,
        redis:    cache.Client,
        stopChan: make(chan struct{}),
    }
}

func (jm *JobManager) StartWorkers() {
    log.Printf("Starting %d job workers", jm.config.Jobs.WorkerCount)
    
    for i := 0; i < jm.config.Jobs.WorkerCount; i++ {
        worker := &Worker{
            id:       fmt.Sprintf("worker-%d", i+1),
            manager:  jm,
            stopChan: make(chan struct{}),
        }
        jm.workers = append(jm.workers, worker)
        go worker.Start()
    }
    
    // Start reminder scheduler
    go jm.startReminderScheduler()
    
    // Start trial expiration checker
    go jm.startTrialExpirationChecker()
    
    log.Println("All job workers started successfully")
}

func (jm *JobManager) Stop() {
    log.Println("Stopping job manager...")
    
    close(jm.stopChan)
    
    for _, worker := range jm.workers {
        close(worker.stopChan)
    }
    
    log.Println("Job manager stopped")
}

func (jm *JobManager) QueueJob(jobType JobType, data map[string]interface{}, scheduledTime *time.Time) (string, error) {
    job := &Job{
        ID:          uuid.New().String(),
        Type:        jobType,
        Data:        data,
        CreatedAt:   time.Now(),
        MaxAttempts: jm.config.Jobs.RetryAttempts,
        Status:      "pending",
    }
    
    if scheduledTime != nil {
        job.ScheduledTime = *scheduledTime
    } else {
        job.ScheduledTime = time.Now()
    }
    
    // Serialize job
    jobData, err := json.Marshal(job)
    if err != nil {
        return "", fmt.Errorf("failed to marshal job: %w", err)
    }
    
    // Add to Redis stream or queue based on scheduling
    ctx := context.Background()
    
    if job.ScheduledTime.After(time.Now()) {
        // Schedule for later
        score := float64(job.ScheduledTime.Unix())
        err = jm.redis.ZAdd(ctx, "scheduled_jobs", &redis.Z{
            Score:  score,
            Member: string(jobData),
        }).Err()
    } else {
        // Queue immediately
        err = jm.redis.LPush(ctx, "job_queue", jobData).Err()
    }
    
    if err != nil {
        return "", fmt.Errorf("failed to queue job: %w", err)
    }
    
    log.Printf("Queued job %s of type %s", job.ID, job.Type)
    return job.ID, nil
}

func (w *Worker) Start() {
    log.Printf("Worker %s started", w.id)
    
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-w.stopChan:
            log.Printf("Worker %s stopped", w.id)
            return
        case <-ticker.C:
            w.processJobs()
        }
    }
}

func (w *Worker) processJobs() {
    ctx := context.Background()
    
    // Check for scheduled jobs that are ready
    w.processScheduledJobs(ctx)
    
    // Process immediate jobs
    w.processImmediateJobs(ctx)
}

func (w *Worker) processScheduledJobs(ctx context.Context) {
    now := time.Now().Unix()
    
    // Get jobs that are ready to be processed
    results, err := w.manager.redis.ZRangeByScoreWithScores(ctx, "scheduled_jobs", &redis.ZRangeBy{
        Min: "0",
        Max: fmt.Sprintf("%d", now),
    }).Result()
    
    if err != nil {
        log.Printf("Worker %s: Error getting scheduled jobs: %v", w.id, err)
        return
    }
    
    for _, result := range results {
        jobData := result.Member.(string)
        
        // Remove from scheduled jobs
        w.manager.redis.ZRem(ctx, "scheduled_jobs", jobData)
        
        // Add to immediate queue
        w.manager.redis.LPush(ctx, "job_queue", jobData)
    }
}

func (w *Worker) processImmediateJobs(ctx context.Context) {
    // Get job from queue (blocking pop with timeout)
    result, err := w.manager.redis.BRPop(ctx, 1*time.Second, "job_queue").Result()
    if err != nil {
        if err != redis.Nil {
            log.Printf("Worker %s: Error popping job: %v", w.id, err)
        }
        return
    }
    
    if len(result) < 2 {
        return
    }
    
    jobData := result[1]
    var job Job
    if err := json.Unmarshal([]byte(jobData), &job); err != nil {
        log.Printf("Worker %s: Error unmarshaling job: %v", w.id, err)
        return
    }
    
    w.executeJob(&job)
}

func (w *Worker) executeJob(job *Job) {
    log.Printf("Worker %s: Processing job %s of type %s", w.id, job.ID, job.Type)
    
    job.Attempts++
    job.Status = "processing"
    
    var err error
    
    switch job.Type {
    case JobEmailNotification:
        err = w.processEmailNotification(job)
    case JobSMSNotification:
        err = w.processSMSNotification(job)
    case JobBookingReminder:
        err = w.processBookingReminder(job)
    case JobAnalyticsUpdate:
        err = w.processAnalyticsUpdate(job)
    case JobTrialExpiration:
        err = w.processTrialExpiration(job)
    case JobFailedPayment:
        err = w.processFailedPayment(job)
    case JobSystemHealthCheck:
        err = w.processSystemHealthCheck(job)
    default:
        err = fmt.Errorf("unknown job type: %s", job.Type)
    }
    
    if err != nil {
        job.Status = "failed"
        job.Error = err.Error()
        
        log.Printf("Worker %s: Job %s failed (attempt %d/%d): %v", 
            w.id, job.ID, job.Attempts, job.MaxAttempts, err)
        
        // Retry if attempts remaining
        if job.Attempts < job.MaxAttempts {
            w.retryJob(job)
        } else {
            log.Printf("Worker %s: Job %s exceeded max attempts, marking as failed", w.id, job.ID)
        }
    } else {
        job.Status = "completed"
        log.Printf("Worker %s: Job %s completed successfully", w.id, job.ID)
    }
    
    // Store job result for tracking
    w.storeJobResult(job)
}

func (w *Worker) retryJob(job *Job) {
    // Calculate delay (exponential backoff)
    delay := time.Duration(job.Attempts) * w.manager.config.Jobs.RetryDelay
    retryTime := time.Now().Add(delay)
    
    job.ScheduledTime = retryTime
    job.Status = "pending"
    
    // Re-queue the job
    jobData, _ := json.Marshal(job)
    ctx := context.Background()
    
    score := float64(retryTime.Unix())
    w.manager.redis.ZAdd(ctx, "scheduled_jobs", &redis.Z{
        Score:  score,
        Member: string(jobData),
    })
    
    log.Printf("Worker %s: Retrying job %s in %v", w.id, job.ID, delay)
}

func (w *Worker) storeJobResult(job *Job) {
    jobData, _ := json.Marshal(job)
    ctx := context.Background()
    
    // Store job result with TTL (keep for 24 hours)
    key := fmt.Sprintf("job_result:%s", job.ID)
    w.manager.redis.Set(ctx, key, jobData, 24*time.Hour)
}

// Job processors
func (w *Worker) processEmailNotification(job *Job) error {
    to, _ := job.Data["to"].(string)
    subject, _ := job.Data["subject"].(string)
    body, _ := job.Data["body"].(string)
    
    if to == "" || subject == "" || body == "" {
        return fmt.Errorf("missing required email data")
    }
    
    // In production, this would use the actual email service
    log.Printf("Sending email to %s: %s", to, subject)
    
    // Simulate email sending delay
    time.Sleep(100 * time.Millisecond)
    
    return nil
}

func (w *Worker) processSMSNotification(job *Job) error {
    to, _ := job.Data["to"].(string)
    message, _ := job.Data["message"].(string)
    
    if to == "" || message == "" {
        return fmt.Errorf("missing required SMS data")
    }
    
    // In production, this would use the actual SMS service
    log.Printf("Sending SMS to %s: %s", to, message)
    
    // Simulate SMS sending delay
    time.Sleep(50 * time.Millisecond)
    
    return nil
}

func (w *Worker) processBookingReminder(job *Job) error {
    bookingID, _ := job.Data["booking_id"].(string)
    
    if bookingID == "" {
        return fmt.Errorf("missing booking ID")
    }
    
    db := database.GetDB()
    
    // Get booking details
    var booking struct {
        ID          string    `db:"id"`
        ClientEmail string    `db:"client_email"`
        ClientPhone string    `db:"client_phone"`
        ClientName  string    `db:"client_name"`
        BookingDate time.Time `db:"booking_date"`
        BookingTime string    `db:"booking_time"`
        Status      string    `db:"status"`
    }
    
    err := db.Get(&booking, `
        SELECT id, client_email, client_phone, client_name, booking_date, booking_time, status
        FROM bookings WHERE id = $1`, bookingID)
    
    if err != nil {
        return fmt.Errorf("failed to get booking: %w", err)
    }
    
    // Don't send reminder for cancelled bookings
    if booking.Status == "cancelled" {
        return nil
    }
    
    // Queue email and SMS notifications
    emailData := map[string]interface{}{
        "to":      booking.ClientEmail,
        "subject": "Appointment Reminder",
        "body":    fmt.Sprintf("Reminder: You have an appointment on %s at %s", 
                   booking.BookingDate.Format("2006-01-02"), booking.BookingTime),
    }
    
    w.manager.QueueJob(JobEmailNotification, emailData, nil)
    
    if booking.ClientPhone != "" {
        smsData := map[string]interface{}{
            "to":      booking.ClientPhone,
            "message": fmt.Sprintf("Reminder: You have an appointment on %s at %s", 
                       booking.BookingDate.Format("2006-01-02"), booking.BookingTime),
        }
        
        w.manager.QueueJob(JobSMSNotification, smsData, nil)
    }
    
    return nil
}

func (w *Worker) processAnalyticsUpdate(job *Job) error {
    // Implement analytics processing
    log.Printf("Processing analytics update")
    
    // This would typically:
    // 1. Calculate daily/weekly/monthly statistics
    // 2. Update cached metrics
    // 3. Generate reports
    
    return nil
}

func (w *Worker) processTrialExpiration(job *Job) error {
    tenantID, _ := job.Data["tenant_id"].(string)
    
    if tenantID == "" {
        return fmt.Errorf("missing tenant ID")
    }
    
    // Process trial expiration logic
    log.Printf("Processing trial expiration for tenant %s", tenantID)
    
    return nil
}

func (w *Worker) processFailedPayment(job *Job) error {
    // Process failed payment logic
    log.Printf("Processing failed payment")
    
    return nil
}

func (w *Worker) processSystemHealthCheck(job *Job) error {
    // Check system health metrics
    log.Printf("Performing system health check")
    
    return nil
}

// Reminder scheduler
func (jm *JobManager) startReminderScheduler() {
    ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
    defer ticker.Stop()
    
    log.Println("Reminder scheduler started")
    
    for {
        select {
        case <-jm.stopChan:
            log.Println("Reminder scheduler stopped")
            return
        case <-ticker.C:
            jm.scheduleBookingReminders()
        }
    }
}

func (jm *JobManager) scheduleBookingReminders() {
    db := database.GetDB()
    
    // Get bookings that need 24h reminders
    reminderTimes := jm.config.Business.ReminderHours
    
    for _, hours := range reminderTimes {
        reminderTime := time.Now().Add(time.Duration(hours) * time.Hour)
        
        var bookings []struct {
            ID                string `db:"id"`
            ReminderSent24h   bool   `db:"reminder_sent_24h"`
            ReminderSent2h    bool   `db:"reminder_sent_2h"`
        }
        
        var columnName string
        if hours == 24 {
            columnName = "reminder_sent_24h"
        } else if hours == 2 {
            columnName = "reminder_sent_2h"
        } else {
            continue // Skip unsupported reminder times
        }
        
        err := db.Select(&bookings, fmt.Sprintf(`
            SELECT id, reminder_sent_24h, reminder_sent_2h
            FROM bookings 
            WHERE booking_date = $1 
            AND status = 'confirmed'
            AND %s = false`, columnName), 
            reminderTime.Format("2006-01-02"))
        
        if err != nil {
            log.Printf("Error getting bookings for reminders: %v", err)
            continue
        }
        
        for _, booking := range bookings {
            // Queue reminder job
            reminderData := map[string]interface{}{
                "booking_id": booking.ID,
            }
            
            jobID, err := jm.QueueJob(JobBookingReminder, reminderData, nil)
            if err != nil {
                log.Printf("Error queueing reminder job: %v", err)
                continue
            }
            
            // Mark reminder as sent
            _, err = db.Exec(fmt.Sprintf(`
                UPDATE bookings SET %s = true WHERE id = $1`, columnName), 
                booking.ID)
            
            if err != nil {
                log.Printf("Error marking reminder as sent: %v", err)
            } else {
                log.Printf("Scheduled %dh reminder job %s for booking %s", 
                    hours, jobID, booking.ID)
            }
        }
    }
}

// Trial expiration checker
func (jm *JobManager) startTrialExpirationChecker() {
    ticker := time.NewTicker(24 * time.Hour) // Check daily
    defer ticker.Stop()
    
    log.Println("Trial expiration checker started")
    
    for {
        select {
        case <-jm.stopChan:
            log.Println("Trial expiration checker stopped")
            return
        case <-ticker.C:
            jm.checkTrialExpirations()
        }
    }
}

func (jm *JobManager) checkTrialExpirations() {
    db := database.GetDB()
    
    // Get tenants with expiring trials
    var tenants []struct {
        ID           string     `db:"id"`
        BusinessName string     `db:"business_name"`
        TrialEndDate *time.Time `db:"trial_end_date"`
        OwnerEmail   string     `db:"owner_email"`
    }
    
    err := db.Select(&tenants, `
        SELECT t.id, t.business_name, t.trial_end_date, u.email as owner_email
        FROM tenants t
        JOIN users u ON t.owner_id = u.id
        WHERE t.status = 'active' 
        AND t.trial_end_date <= $1`, 
        time.Now().Add(7*24*time.Hour)) // Check 7 days in advance
    
    if err != nil {
        log.Printf("Error getting expiring trials: %v", err)
        return
    }
    
    for _, tenant := range tenants {
        if tenant.TrialEndDate == nil {
            continue
        }
        
        daysUntilExpiry := int(time.Until(*tenant.TrialEndDate).Hours() / 24)
        
        if daysUntilExpiry <= 7 && daysUntilExpiry >= 0 {
            // Queue trial expiration notification
            notificationData := map[string]interface{}{
                "tenant_id":     tenant.ID,
                "business_name": tenant.BusinessName,
                "owner_email":   tenant.OwnerEmail,
                "days_remaining": daysUntilExpiry,
            }
            
            _, err := jm.QueueJob(JobTrialExpiration, notificationData, nil)
            if err != nil {
                log.Printf("Error queueing trial expiration job: %v", err)
            } else {
                log.Printf("Queued trial expiration notification for tenant %s (%d days)", 
                    tenant.ID, daysUntilExpiry)
            }
        }
    }
}
