# Crossonic Server

Crossonic-Server is a Subsonic-compatible music server with the aim to support modern features of the [OpenSubsonic API](https://opensubsonic.netlify.app/) and custom extensions
for use with the [Crossonic client application](https://crossonic.org/app).

It works by scanning a directory containing your music files and making your music library available to [Crossonic](https://crossonic.org/app)
and all other (Open)Subsonic compatible clients, that can be used to browse
and stream your music on all your devices similar to popular streaming services.

## Installation 

Follow the [installation guide](https://crossonic.org/server/install).

## Features
- Robust library scanning
    - Takes MusicBrainz IDs into account
    - Stores crossonic song ID in the file metadata to prevent losing favorites/scrobbles when renaming files and/or changing metadata
    - Multiple artists/genres per song
    - Release groups, labels, disc subtitles, replay gain, lyrics, bpm, …
    - Incremental scanning (only scans files that have changed)
- Multi-user
    - Each with their own playlists, scrobbles, favorites, …
    - Add internet radio stations per user
- ListenBrainz integration
    - Scrobbling
    - Two-way favorites sync
    - Configurable for each user
- Fetch artist images and biographies from [last.fm](https://last.fm)
- On-the-fly transcoding and caching
    - Configurable `format=` and `maxBitRate=` parameters
    - `raw`, `mp3`, `opus`, `vorbis`
- Receive scrobbles from clients (including playback duration)
- Proper handling of different release versions
- Compatible with your favorite (Open)Subsonic clients

## Documentation

All documentation can be found on the [Crossonic website](https://crossonic.org/server).

Some useful links:
- [Installation](https://crossonic.org/server/install)
- [Configuration](https://crossonic.org/server/configuration)
- [Music library organization](https://crossonic.org/server/music-library)

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
