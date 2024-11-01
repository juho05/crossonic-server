# Crossonic Server

OpenSubsonic compatible music server with additional extensions for [Crossonic](https://github.com/juho05/crossonic).

## Status

This project is in development. Expect bugs and missing features.

Not all OpenSubsonic endpoints have been implemented yet ([status](./supported_endpoints.md)).
This server implements all endpoints needed for the [Crossonic](https://github.com/juho05/crossonic) app but may not work with
other Subsonic media players.

## Features

- [x] Full scan of media library (*kind of slow currently*)
- [ ] Incremental library scan
- [x] [ListenBrainz](https://listenbrainz.org) scrobbling
- [x] Multiple users
- [x] transcoding and caching
  - [x] configurable with `format=` and `maxBitRate=` parameters
  - `raw`, `mp3`, `opus`, `vorbis`, `aac`
- [x] Fetch artist images from [last.fm](https://last.fm)
- [x] Multiple artists, album artists, genres
- [x] **Stores a unique ID in the metadata of all media files** to preserve IDs when moving/renaming files on disk
- [x] Scrobbling including playback duration
- [x] Browse by tags
  - browsing by folders not supported
- [x] Favorites/rating
- [x] Lyrics
  - [x] unsynced
  - [ ] synced
- [x] playlists
  - including user-changable playlist covers (not natively supported by *OpenSubsonic*)
- [x] [SONOS](https://www.sonos.com) casting (*very buggy*, *not documented*)
- [x] Serve [Crossonic](https://github.com/juho05/crossonic#web)
- [x] Admin CLI
- [ ] Admin web interface

## Deploy with Docker

Create `docker-compose.yml`:
```bash
services:
  crossonic:
    image: ghcr.io/juho05/crossonic-server
    restart: unless-stopped
    environment:
      DB_USER: crossonic
      DB_PASSWORD: crossonic
      DB_HOST: db
      DB_PORT: 5432
      DB_NAME: crossonic
      # Base64 encoded string representing exactly 32 bytes.
      # Generate with: docker run --rm -it --entrypoint crossonic-admin ghcr.io/juho05/crossonic-server gen-encryption-key
      PASSWORD_ENCRYPTION_KEY: <key>
      # URL where crossonic-server is reachable
      BASE_URL: "https://crossonic.example.com"
    volumes:
      - "./cache:/cache"   # cache files
      - "./data:/data"     # cover art etc.
      - "./library:/music" # your music files
    ports:
      - "8080:8080"
    depends_on:
      - db
  db:
    image: postgres:16
    restart: unless-stopped
    volumes:
      - ./postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_PASSWORD: crossonic
      POSTGRES_USER: crossonic
    healthcheck:
      test: ["CMD", "pg_isready -U crossonic"]
      interval: 30s
      timeout: 20s
      retries: 3
```

Run `docker compose up -d` in the same directory as the `docker-compose.yml` file.

### Create user

To create a user use the following command from the directory of your `docker-compose.yml` file:

```bash
docker compose exec -it crossonic crossonic-admin user create <name> <password>
```