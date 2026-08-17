# Stage 1: Build Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

# Stage 2: Final image
FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /server .
COPY --from=builder /app/package.json .
EXPOSE 4000
VOLUME ["/app/data"]
CMD ["./server"]
