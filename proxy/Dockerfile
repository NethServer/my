FROM nginx:1.25-alpine

# Install curl for health checks
RUN apk add --no-cache curl

# Copy nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Create log directory
RUN mkdir -p /var/log/nginx

# Expose port (Render assigns dynamically)
EXPOSE $PORT

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:$PORT/health || exit 1

# Start nginx
CMD ["sh", "-c", "envsubst '$$PORT $$BACKEND_SERVICE_NAME $$COLLECT_SERVICE_NAME $$FRONTEND_SERVICE_NAME' < /etc/nginx/nginx.conf > /tmp/nginx.conf && nginx -c /tmp/nginx.conf -g 'daemon off;'"]