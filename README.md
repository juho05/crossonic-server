# Crossonic Server

OpenSubsonic compatible music server with additional extensions for [Crossonic](https://github.com/juho05/crossonic).

## Status

This project is in development. Expect bugs and missing features.

Not all OpenSubsonic endpoints have been implemented yet ([status](./supported_endpoints.md)).
This server implements all endpoints needed for the [Crossonic](https://github.com/juho05/crossonic) app but may not yet work without issues with
other Subsonic media players.

## Features

- [x] media library scan
  - [x] on startup
  - [x] `startScan` endpoint
  - [x] incremental scan
  - [ ] file watching
- [x] [ListenBrainz](https://listenbrainz.org) integration
  - [x] scrobbling
  - [x] sync favorites
- [x] Multiple users
  - [x] per-user internet radio stations
- [x] Transcoding and caching
  - [x] configurable with `format=` and `maxBitRate=` parameters
  - `raw`, `mp3`, `opus`, `vorbis`
- [x] Fetch artist images and artist/album info from [last.fm](https://last.fm)
- [x] Multiple artists, album artists, genres
- [x] **Stores a unique ID in the metadata of all media files** to preserve IDs when moving/renaming files on disk
- [x] Scrobbling including playback duration
- [x] Browse by tags
  - browsing by folders not supported (`getIndexes`, `getMusicDirectory` etc. simulate behavior based on tags)
- [x] Favorites/rating
- [x] Lyrics
  - [x] unsynced
  - [ ] synced
- [ ] Multiple music directories
- [x] Playlists
  - including user-changable playlist covers (not natively supported by *OpenSubsonic*)
- [ ] Jukebox
  - [ ] device selection
  - [ ] SONOS casting
- [x] Serve [Crossonic Web](https://github.com/juho05/crossonic#web)
- [x] Admin CLI
- [ ] Admin web interface
- [x] End-of-year recap
  - [x] total listening duration
  - [x] distinct song, album, artist count
  - [ ] ranked songs, albums, artists by listening duration

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
      ENCRYPTION_KEY: <key>
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
docker compose exec -it crossonic crossonic-admin users create <name>
```

## License

Copyright (c) 2024-2025 Julian Hofmann

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
