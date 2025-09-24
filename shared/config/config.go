package config

import (
    "os"
    "strconv"
    "strings"
    "time"
)

type Config struct {
    // Environment
    Environment    string
    
    // Domain Configuration
    BaseDomain     string
    MainDomain     string
    AdminDomain    string
    
    // Database Configuration
    Database DatabaseConfig
    
    // Redis Configuration
    Redis RedisConfig
    
    // JWT Configuration
    JWT JWTConfig
    
    // Service Ports
    Services ServiceConfig
    
    // Email Configuration
    Email EmailConfig
    
    // SMS Configuration
    SMS SMSConfig
    
    // Security Configuration
    Security SecurityConfig
    
    // Background Jobs
    Jobs JobConfig
    
    // Business Logic
    Business BusinessConfig
    
    // Internationalization
    I18n I18nConfig
    
    // SSL Configuration
    SSL SSLConfig
}

type DatabaseConfig struct {
    Host           string
    Port           int
    Name           string
    User           string
    Password       string
    MaxConnections int
    MaxIdle        int
}

type RedisConfig struct {
    Host           string
    Port           int
    Password       string
    DB             int
    MaxConnections int
}

type JWTConfig struct {
    Secret        string
    Expiry        time.Duration
    RefreshExpiry time.Duration
}

type ServiceConfig struct {
    APIGateway    int
    UserHTTP      int
    UserGRPC      int
    BookingHTTP   int
    BookingGRPC   int
    NotificationHTTP int
    NotificationGRPC int
    PaymentHTTP   int
    PaymentGRPC   int
    AdminHTTP     int
    AdminGRPC     int
}

type EmailConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    From     string
}

type SMSConfig struct {
    Provider  string
    APIKey    string
    APISecret string
}

type SecurityConfig struct {
    BCryptCost        int
    RateLimitRequests int
    RateLimitWindow   time.Duration
    CORSOrigins       []string
}

type JobConfig struct {
    WorkerCount    int
    RetryAttempts  int
    RetryDelay     time.Duration
}

type BusinessConfig struct {
    DefaultTrialDays    int
    BookingAdvanceLimit time.Duration
    CancellationHours   int
    ReminderHours       []int
}

type I18nConfig struct {
    DefaultLanguage     string
    SupportedLanguages  []string
}

type SSLConfig struct {
    CertPath string
    KeyPath  string
}

func Load() *Config {
    config := &Config{
        Environment: getEnv("ENVIRONMENT", "development"),
        BaseDomain:  getEnv("BASE_DOMAIN", "jazyl.tech"),
        MainDomain:  getEnv("MAIN_DOMAIN", "jazyl.tech"),
        AdminDomain: getEnv("ADMIN_DOMAIN", "admin.jazyl.tech"),
        
        Database: DatabaseConfig{
            Host:           getEnv("DB_HOST", "localhost"),
            Port:           getEnvInt("DB_PORT", 5432),
            Name:           getEnv("DB_NAME", "booking_platform"),
            User:           getEnv("DB_USER", "booking_user"),
            Password:       getEnv("DB_PASSWORD", ""),
            MaxConnections: getEnvInt("DB_MAX_CONNECTIONS", 100),
            MaxIdle:        getEnvInt("DB_MAX_IDLE", 10),
        },
        
        Redis: RedisConfig{
            Host:           getEnv("REDIS_HOST", "localhost"),
            Port:           getEnvInt("REDIS_PORT", 6379),
            Password:       getEnv("REDIS_PASSWORD", ""),
            DB:             getEnvInt("REDIS_DB", 0),
            MaxConnections: getEnvInt("REDIS_MAX_CONNECTIONS", 50),
        },
        
        JWT: JWTConfig{
            Secret:        getEnv("JWT_SECRET", "your_jwt_secret_key"),
            Expiry:        getEnvDuration("JWT_EXPIRY", 24*time.Hour),
            RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 168*time.Hour),
        },
        
        Services: ServiceConfig{
            APIGateway:       getEnvInt("API_GATEWAY_PORT", 8080),
            UserHTTP:         getEnvInt("USER_SERVICE_PORT", 8081),
            UserGRPC:         getEnvInt("USER_SERVICE_GRPC_PORT", 50051),
            BookingHTTP:      getEnvInt("BOOKING_SERVICE_PORT", 8082),
            BookingGRPC:      getEnvInt("BOOKING_SERVICE_GRPC_PORT", 50052),
            NotificationHTTP: getEnvInt("NOTIFICATION_SERVICE_PORT", 8083),
            NotificationGRPC: getEnvInt("NOTIFICATION_SERVICE_GRPC_PORT", 50053),
            PaymentHTTP:      getEnvInt("PAYMENT_SERVICE_PORT", 8084),
            PaymentGRPC:      getEnvInt("PAYMENT_SERVICE_GRPC_PORT", 50054),
            AdminHTTP:        getEnvInt("ADMIN_SERVICE_PORT", 8085),
            AdminGRPC:        getEnvInt("ADMIN_SERVICE_GRPC_PORT", 50055),
        },
        
        Email: EmailConfig{
            Host:     getEnv("EMAIL_HOST", "smtp.gmail.com"),
            Port:     getEnvInt("EMAIL_PORT", 587),
            User:     getEnv("EMAIL_USER", ""),
            Password: getEnv("EMAIL_PASSWORD", ""),
            From:     getEnv("EMAIL_FROM", "noreply@jazyl.tech"),
        },
        
        SMS: SMSConfig{
            Provider:  getEnv("SMS_PROVIDER", "twilio"),
            APIKey:    getEnv("SMS_API_KEY", ""),
            APISecret: getEnv("SMS_API_SECRET", ""),
        },
        
        Security: SecurityConfig{
            BCryptCost:        getEnvInt("BCRYPT_COST", 12),
            RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
            RateLimitWindow:   getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
            CORSOrigins:       strings.Split(getEnv("CORS_ORIGINS", "*.jazyl.tech"), ","),
        },
        
        Jobs: JobConfig{
            WorkerCount:   getEnvInt("WORKER_COUNT", 5),
            RetryAttempts: getEnvInt("JOB_RETRY_ATTEMPTS", 3),
            RetryDelay:    getEnvDuration("JOB_RETRY_DELAY", 30*time.Second),
        },
        
        Business: BusinessConfig{
            DefaultTrialDays:    getEnvInt("DEFAULT_TRIAL_DAYS", 30),
            BookingAdvanceLimit: getEnvDuration("BOOKING_ADVANCE_LIMIT", 30*24*time.Hour),
            CancellationHours:   getEnvInt("CANCELLATION_HOURS", 2),
            ReminderHours:       parseIntSlice(getEnv("REMINDER_HOURS", "24,2")),
        },
        
        I18n: I18nConfig{
            DefaultLanguage:    getEnv("DEFAULT_LANGUAGE", "en"),
            SupportedLanguages: strings.Split(getEnv("SUPPORTED_LANGUAGES", "en,ru,kk"), ","),
        },
        
        SSL: SSLConfig{
            CertPath: getEnv("SSL_CERT_PATH", "./ssl/jazyl.tech.pem"),
            KeyPath:  getEnv("SSL_KEY_PATH", "./ssl/jazyl.tech.key"),
        },
    }
    
    return config
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

func parseIntSlice(value string) []int {
    parts := strings.Split(value, ",")
    result := make([]int, 0, len(parts))
    for _, part := range parts {
        if intValue, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
            result = append(result, intValue)
        }
    }
    return result
}
