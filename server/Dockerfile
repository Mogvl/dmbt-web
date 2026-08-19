# dmbt-web backend image: Go API server (port 9701).

FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata su-exec \
    && addgroup -S animegarden && adduser -S animegarden -G animegarden

WORKDIR /app
COPY --from=build /out/server /app/server
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

# data directory (bind-mount /data in compose); ownership is fixed by the
# entrypoint so host-created bind mounts (root-owned) work out of the box.
ENV PORT=9701 \
    DATA_DIR=/data \
    CRON=true \
    TZ=Asia/Shanghai

EXPOSE 9701

ENTRYPOINT ["/entrypoint.sh"]
