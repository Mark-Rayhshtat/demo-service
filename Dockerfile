FROM golang:1.21-alpine as builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Create a minimal image
FROM alpine:latest  

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/app .

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["./app"] 