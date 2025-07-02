# Build stage
FROM golang:1.24.4-alpine3.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o rant cmd/rant-server

# Runtime stage
FROM alpine:3.22

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/cmd/rant-server/rant .

EXPOSE 8080

CMD ["./rant"]