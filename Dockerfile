# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/dhlottery .

# HTTPS(동행복권/텔레그램) + Asia/Seoul 스케줄에 ca-certificates, tzdata 필요
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Seoul

WORKDIR /app

COPY --from=builder /out/dhlottery /app/dhlottery

RUN mkdir -p /app/logs

# config.json, logs 는 compose volume 로 마운트
ENTRYPOINT ["/app/dhlottery"]
CMD ["-service"]
