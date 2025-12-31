FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o go-bot main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai

COPY --from=builder /app/go-bot .

# Copy config folder if needed, but config.go loads from env or .env
# We assume .env is mounted or env vars are passed

CMD ["./go-bot"]
