# Build Stage
FROM golang:1.23.4-alpine AS builder

LABEL authors="the-eduardo"

WORKDIR /app

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o /go/bin/app ./...

# Final stage
FROM arm64v8/alpine:3.19

WORKDIR /app

# Copy only the built binary from the builder stage
COPY --from=builder /go/bin/app /app/

# Command to run the executable
CMD ["./app"]