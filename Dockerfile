FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=1 is required for go-sqlite3
RUN apk add --no-cache gcc musl-dev
RUN CGO_ENABLED=1 GOOS=linux go build -a -o vod-server ./cmd/vod/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/vod-server .
COPY --from=builder /app/frontend ./frontend

EXPOSE 8080
CMD ["./vod-server"]
