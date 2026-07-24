FROM golang:1.25.12-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o echosvr main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/echosvr .
COPY config.yml .

EXPOSE 58080 58081

CMD ["./echosvr"]
