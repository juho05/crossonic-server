# docker buildx build --platform linux/arm64,linux/amd64 --tag ghcr.io/juho05/crossonic-server:latest --push .
FROM alpine:3.19 AS builder-taglib
WORKDIR /tmp
COPY alpine/taglib/APKBUILD .
RUN apk update && \
    apk add --no-cache abuild && \
    abuild-keygen -a -n && \
    REPODEST=/pkgs abuild -F -r

FROM golang:alpine3.19 AS builder
RUN apk add -U --no-cache \
    build-base \
    ca-certificates \
    git \
    zlib-dev

# TODO: delete this block when taglib v2 is on alpine packages
COPY --from=builder-taglib /pkgs/*/*.apk /pkgs/
RUN apk add --no-cache --allow-untrusted /pkgs/*

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GOOS=linux go build -o crossonic-server ./cmd/server
RUN GOOS=linux go build -o crossonic-admin ./cmd/admin

FROM alpine:3.19
LABEL org.opencontainers.image.source https://github.com/juho05/crossonic-server
RUN apk add -U --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \
    tini \
    shared-mime-info

COPY --from=builder \
    /usr/lib/libgcc_s.so.1 \
    /usr/lib/libstdc++.so.6 \
    /usr/lib/libtag.so.2 \
    /usr/lib/
COPY --from=builder \
    /src/crossonic-server \
    /src/crossonic-admin \
    /bin/
EXPOSE 8080
ENV TZ ""
ENV MUSIC_DIR /music
ENV PLAYLISTS_DIR /playlists
ENV DATA_DIR /data
ENV LISTEN_ADDR 0.0.0.0:8080
ENV AUTO_MIGRATE true
ENV LOG_LEVEL 4
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["crossonic-server"]