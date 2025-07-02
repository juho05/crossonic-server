# docker buildx build --platform linux/arm64,linux/amd64 --tag ghcr.io/juho05/crossonic-server:latest --push .
FROM golang:alpine3.22 AS builder
RUN apk add -U --no-cache \
    build-base \
    ca-certificates \
    git \
    zlib-dev \
    taglib-dev

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GOOS=linux go build -o crossonic-server ./cmd/server
RUN GOOS=linux go build -o crossonic-admin ./cmd/admin

FROM alpine:3.22
LABEL org.opencontainers.image.source=https://github.com/juho05/crossonic-server
RUN apk add -U --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \
    tini \
    shared-mime-info \
    taglib

COPY --from=builder "/usr/lib/libgcc_s.so.1" /usr/lib/
COPY --from=builder "/usr/lib/libstdc++.so.6" /usr/lib/
COPY --from=builder "/usr/lib/libtag.so.2" /usr/lib/

COPY --from=builder "/src/crossonic-server" /bin/
COPY --from=builder "/src/crossonic-admin" /bin/
EXPOSE 8080
ENV TZ=""
ENV MUSIC_DIR=/music
ENV DATA_DIR=/data
ENV CACHE_DIR=/cache
ENV LISTEN_ADDR=0.0.0.0:8080
ENV AUTO_MIGRATE=true
ENV LOG_LEVEL=4
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["crossonic-server"]