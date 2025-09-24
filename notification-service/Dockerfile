FROM booking-platform-base:latest AS builder

# Copy only this service's code
COPY notification-service/ ./notification-service/

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o main ./notification-service

# Production stage
FROM alpine:3.18

# Install only necessary packages
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary and assets
COPY --from=builder /app/main .
COPY --from=builder /app/shared/i18n/locales ./shared/i18n/locales
COPY --from=builder /app/notification-service/templates ./notification-service/templates

# Change ownership
RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8083 50053

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD curl -f http://localhost:8083/health || exit 1

CMD ["./main"]
