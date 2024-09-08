# Dockerfile for ERP service
FROM golang:latest

# Download Go modules
COPY services/erp/go.mod services/erp/go.sum ./
RUN go mod download

# Copy code
COPY services/erp/*.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /main

EXPOSE 1323

# Run
CMD ["/main"]
