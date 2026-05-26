# syntax=docker/dockerfile:1

FROM golang:1.25.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gameserver ./cmd/gameserver

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& addgroup -S app \
	&& adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/gameserver ./gameserver
COPY etc ./etc

USER app

EXPOSE 8088

CMD ["./gameserver", "-f", "etc/gameserver.yaml"]
