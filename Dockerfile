FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o billing-app ./cmd/billing/main.go
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/billing-app .
COPY --from=builder /app/config.yaml .
EXPOSE 8080
CMD ["./billing-app"]