package services

import (
    "context"
    "crypto/tls"
    "fmt"
    "net/smtp"
    "strings"
    
    "github.com/google/uuid"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/config"
    "booking-platform/shared/i18n"
    "booking-platform/notification-service/templates"
    pb "booking-platform/notification-service/proto"
)

type NotificationService struct {
    pb.UnimplementedNotificationServiceServer
    config      *config.Config
    emailClient *EmailClient
    smsClient   *SMSClient
}

type EmailClient struct {
    host     string
    port     int
    username string
    password string
    from     string
}

type SMSClient struct {
    provider  string
    apiKey    string
    apiSecret string
}

func NewNotificationService(cfg *config.Config) *NotificationService {
    emailClient := &EmailClient{
        host:     cfg.Email.Host,
        port:     cfg.Email.Port,
        username: cfg.Email.User,
        password: cfg.Email.Password,
        from:     cfg.Email.From,
    }
    
    smsClient := &SMSClient{
        provider:  cfg.SMS.Provider,
        apiKey:    cfg.SMS.APIKey,
        apiSecret: cfg.SMS.APISecret,
    }
    
    return &NotificationService{
        config:      cfg,
        emailClient: emailClient,
        smsClient:   smsClient,
    }
}

func (s *NotificationService) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
    var body string
    var err error
    
    if req.TemplateName != "" {
        // Use template
        body, err = s.renderTemplate(req.TemplateName, req.TemplateData, req.Language)
        if err != nil {
            return nil, status.Error(codes.Internal, "Failed to render template")
        }
    } else {
        body = req.Body
    }
    
    messageID, err := s.emailClient.SendEmail(req.To, req.Subject, body)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to send email")
    }
    
    return &pb.SendEmailResponse{
        MessageId: messageID,
        Message:   "Email sent successfully",
    }, nil
}

func (s *NotificationService) SendBookingConfirmation(ctx context.Context, req *pb.SendBookingConfirmationRequest) (*pb.SendBookingConfirmationResponse, error) {
    templateData := map[string]string{
        "client_name":       req.ClientName,
        "business_name":     req.BusinessName,
        "service_name":      req.ServiceName,
        "master_name":       req.MasterName,
        "booking_date":      req.BookingDate,
        "booking_time":      req.BookingTime,
        "location_address":  req.LocationAddress,
        "confirmation_code": req.ConfirmationCode,
    }
    
    subject := i18n.T(req.Language, "email.booking.confirmation.subject", req.BusinessName)
    body, err := s.renderTemplate("booking_confirmation", templateData, req.Language)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to render template")
    }
    
    messageID, err := s.emailClient.SendEmail(req.ClientEmail, subject, body)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to send confirmation email")
    }
    
    return &pb.SendBookingConfirmationResponse{
        MessageId: messageID,
        Message:   "Booking confirmation sent successfully",
    }, nil
}

func (s *NotificationService) SendBookingReminder(ctx context.Context, req *pb.SendBookingReminderRequest) (*pb.SendBookingReminderResponse, error) {
    templateData := map[string]string{
        "client_name":      req.ClientName,
        "business_name":    req.BusinessName,
        "service_name":     req.ServiceName,
        "master_name":      req.MasterName,
        "booking_date":     req.BookingDate,
        "booking_time":     req.BookingTime,
        "location_address": req.LocationAddress,
        "hours_before":     fmt.Sprintf("%d", req.HoursBefore),
    }
    
    var emailMessageID, smsMessageID string
    
    // Send email reminder
    subject := i18n.T(req.Language, "email.booking.reminder.subject", req.BusinessName)
    body, err := s.renderTemplate("booking_reminder", templateData, req.Language)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to render email template")
    }
    
    emailMessageID, err = s.emailClient.SendEmail(req.ClientEmail, subject, body)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to send reminder email")
    }
    
    // Send SMS reminder if phone number is provided
    if req.ClientPhone != "" {
        smsMessage := i18n.T(req.Language, "sms.booking.reminder", req.BookingTime, req.BookingDate)
        smsMessageID, err = s.smsClient.SendSMS(req.ClientPhone, smsMessage)
        if err != nil {
            // Log error but don't fail the entire operation
            fmt.Printf("Failed to send SMS reminder: %v\n", err)
        }
    }
    
    return &pb.SendBookingReminderResponse{
        EmailMessageId: emailMessageID,
        SmsMessageId:   smsMessageID,
        Message:        "Booking reminder sent successfully",
    }, nil
}

func (s *NotificationService) SendBookingCancellation(ctx context.Context, req *pb.SendBookingCancellationRequest) (*pb.SendBookingCancellationResponse, error) {
    templateData := map[string]string{
        "client_name":         req.ClientName,
        "business_name":       req.BusinessName,
        "service_name":        req.ServiceName,
        "booking_date":        req.BookingDate,
        "booking_time":        req.BookingTime,
        "cancellation_reason": req.CancellationReason,
    }
    
    subject := i18n.T(req.Language, "email.booking.cancellation.subject", req.BusinessName)
    body, err := s.renderTemplate("booking_cancellation", templateData, req.Language)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to render template")
    }
    
    messageID, err := s.emailClient.SendEmail(req.ClientEmail, subject, body)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to send cancellation email")
    }
    
    return &pb.SendBookingCancellationResponse{
        MessageId: messageID,
        Message:   "Booking cancellation sent successfully",
    }, nil
}

func (s *NotificationService) SendSMS(ctx context.Context, req *pb.SendSMSRequest) (*pb.SendSMSResponse, error) {
    messageID, err := s.smsClient.SendSMS(req.To, req.Message)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to send SMS")
    }
    
    return &pb.SendSMSResponse{
        MessageId: messageID,
        Message:   "SMS sent successfully",
    }, nil
}

func (s *NotificationService) SendVerificationCode(ctx context.Context, req *pb.SendVerificationCodeRequest) (*pb.SendVerificationCodeResponse, error) {
    message := i18n.T(req.Language, "sms.verification.code", req.Code)
    
    messageID, err := s.smsClient.SendSMS(req.Phone, message)
    if err != nil {
        return nil, status.Error(codes.Internal, "Failed to send verification code")
    }
    
    return &pb.SendVerificationCodeResponse{
        MessageId: messageID,
        Message:   "Verification code sent successfully",
    }, nil
}

// Email client implementation
func (e *EmailClient) SendEmail(to, subject, body string) (string, error) {
    // Create message
    messageID := uuid.New().String()
    
    msg := fmt.Sprintf("To: %s\r\n", to) +
        fmt.Sprintf("From: %s\r\n", e.from) +
        fmt.Sprintf("Subject: %s\r\n", subject) +
        fmt.Sprintf("Message-ID: <%s@%s>\r\n", messageID, strings.Split(e.from, "@")[1]) +
        "Content-Type: text/html; charset=UTF-8\r\n\r\n" +
        body
    
    // Connect to SMTP server
    auth := smtp.PlainAuth("", e.username, e.password, e.host)
    
    // For Gmail and other TLS-required servers
    tlsConfig := &tls.Config{
        InsecureSkipVerify: false,
        ServerName:         e.host,
    }
    
    conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", e.host, e.port), tlsConfig)
    if err != nil {
        return "", fmt.Errorf("failed to connect to SMTP server: %w", err)
    }
    defer conn.Close()
    
    client, err := smtp.NewClient(conn, e.host)
    if err != nil {
        return "", fmt.Errorf("failed to create SMTP client: %w", err)
    }
    defer client.Quit()
    
    if err := client.Auth(auth); err != nil {
        return "", fmt.Errorf("SMTP auth failed: %w", err)
    }
    
    if err := client.Mail(e.from); err != nil {
        return "", fmt.Errorf("failed to set sender: %w", err)
    }
    
    if err := client.Rcpt(to); err != nil {
        return "", fmt.Errorf("failed to set recipient: %w", err)
    }
    
    writer, err := client.Data()
    if err != nil {
        return "", fmt.Errorf("failed to get data writer: %w", err)
    }
    
    if _, err := writer.Write([]byte(msg)); err != nil {
        return "", fmt.Errorf("failed to write message: %w", err)
    }
    
    if err := writer.Close(); err != nil {
        return "", fmt.Errorf("failed to close writer: %w", err)
    }
    
    return messageID, nil
}

// SMS client implementation (simplified - would use actual SMS provider API)
func (s *SMSClient) SendSMS(to, message string) (string, error) {
    // This is a simplified implementation
    // In production, you would integrate with actual SMS providers like Twilio, AWS SNS, etc.
    
    messageID := uuid.New().String()
    
    // Log SMS for development
    fmt.Printf("SMS to %s: %s (MessageID: %s)\n", to, message, messageID)
    
    // In production, implement actual SMS sending logic here
    // Example for Twilio:
    // - Make HTTP POST to Twilio API
    // - Handle authentication and response parsing
    // - Return actual message ID from provider
    
    return messageID, nil
}

// Template rendering
func (s *NotificationService) renderTemplate(templateName string, data map[string]string, language string) (string, error) {
    template, exists := templates.GetTemplate(templateName, language)
    if !exists {
        // Fall back to default language
        template, exists = templates.GetTemplate(templateName, s.config.I18n.DefaultLanguage)
        if !exists {
            return "", fmt.Errorf("template not found: %s", templateName)
        }
    }
    
    // Simple template variable replacement
    result := template
    for key, value := range data {
        placeholder := fmt.Sprintf("{{%s}}", key)
        result = strings.ReplaceAll(result, placeholder, value)
    }
    
    return result, nil
}

// Placeholder implementations for other methods
func (s *NotificationService) QueueJob(ctx context.Context, req *pb.QueueJobRequest) (*pb.QueueJobResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *NotificationService) GetJobStatus(ctx context.Context, req *pb.GetJobStatusRequest) (*pb.GetJobStatusResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *NotificationService) SendBulkEmails(ctx context.Context, req *pb.SendBulkEmailsRequest) (*pb.SendBulkEmailsResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

func (s *NotificationService) SendBulkSMS(ctx context.Context, req *pb.SendBulkSMSRequest) (*pb.SendBulkSMSResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}
