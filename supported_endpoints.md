# List of Supported Endpoints

## OpenSubsonic (1.16.1)

**Response formats:** `xml`, `json`

### System

- [x] [ping](https://opensubsonic.netlify.app/docs/endpoints/ping)
- [x] [getLicense](https://opensubsonic.netlify.app/docs/endpoints/getlicense) (*only `valid` field*)
- [ ] [tokenInfo](https://opensubsonic.netlify.app/docs/endpoints/tokeninfo/)
- [x] [getOpenSubsonicExtensions](https://opensubsonic.netlify.app/docs/endpoints/getopensubsonicextensions)
  - [x] [HTTP form POST](https://opensubsonic.netlify.app/docs/extensions/formpost/)
  - [x] [Transcode Offset](https://opensubsonic.netlify.app/docs/extensions/transcodeoffset/)
  - [x] [Song Lyrics](https://opensubsonic.netlify.app/docs/extensions/songlyrics/) (*only unsynced*)
  - [ ] [API Key Authentication](https://opensubsonic.netlify.app/docs/extensions/apikeyauth/)

### Browsing

- [ ] [getMusicFolders](https://opensubsonic.netlify.app/docs/endpoints/getmusicfolders)
- [x] [getIndexes](https://opensubsonic.netlify.app/docs/endpoints/getindexes)
- [x] [getMusicDirectory](https://opensubsonic.netlify.app/docs/endpoints/getmusicdirectory)
  - supports artist and album IDs
- [x] [getGenres](https://opensubsonic.netlify.app/docs/endpoints/getgenres)
- [x] [getArtists](https://opensubsonic.netlify.app/docs/endpoints/getartists)
  - only album artists are returned
- [x] [getArtist](https://opensubsonic.netlify.app/docs/endpoints/getartist)
- [x] [getAlbum](https://opensubsonic.netlify.app/docs/endpoints/getalbum)
- [x] [getSong](https://opensubsonic.netlify.app/docs/endpoints/getsong)
- [x] [getArtistInfo](https://opensubsonic.netlify.app/docs/endpoints/getartistinfo)
  - identical to _getArtistInfo2_
- [x] [getArtistInfo2](https://opensubsonic.netlify.app/docs/endpoints/getartistinfo2)
  - [ ] similar artists
- [x] [getAlbumInfo](https://opensubsonic.netlify.app/docs/endpoints/getalbuminfo)
  - identical to _getAlbumInfo2_
- [x] [getAlbumInfo2](https://opensubsonic.netlify.app/docs/endpoints/getalbuminfo2)
  - [ ] similar albums
- [ ] [getSimilarSongs](https://opensubsonic.netlify.app/docs/endpoints/getsimilarsongs)
- [ ] [getSimilarSongs2](https://opensubsonic.netlify.app/docs/endpoints/getsimilarsongs2)
- [ ] [getTopSongs](https://opensubsonic.netlify.app/docs/endpoints/gettopsongs)

### Album/Song Lists

- [x] [getAlbumList](https://opensubsonic.netlify.app/docs/endpoints/getalbumlist)
  - identical to _getAlbumList2_
- [x] [getAlbumList2](https://opensubsonic.netlify.app/docs/endpoints/getalbumlist2)
  - [x] random
  - [x] newest
  - [x] highest
  - [x] frequent
  - [x] recent
  - [x] alphabeticalByName
  - [ ] alphabeticalByArtist -> *impossible because of multiple artists*
  - [x] starred
  - [x] byYear
  - [x] byGenre
- [x] [getRandomSongs](https://opensubsonic.netlify.app/docs/endpoints/getrandomsongs)
- [x] [getSongsByGenre](https://opensubsonic.netlify.app/docs/endpoints/getsongsbygenre)
- [x] [getNowPlaying](https://opensubsonic.netlify.app/docs/endpoints/getnowplaying)
- [x] [getStarred](https://opensubsonic.netlify.app/docs/endpoints/getstarred)
  - identical to _getStarred2_
- [x] [getStarred2](https://opensubsonic.netlify.app/docs/endpoints/getstarred2)
  - (optional) pagination via `songLimit`, `songOffset`, `albumLimit`, `albumOffset`, `artistLimit`, `artistOffset` (similar to `search3`)

### Searching

- [x] [search2](https://opensubsonic.netlify.app/docs/endpoints/search2)
  - identical to _search3_
- [x] [search3](https://opensubsonic.netlify.app/docs/endpoints/search3)
  - only album artists are returned

### Playlists

*public playlists are disabled*

- [x] [getPlaylists](https://opensubsonic.netlify.app/docs/endpoints/getplaylists)
- [x] [getPlaylist](https://opensubsonic.netlify.app/docs/endpoints/getplaylist)
- [x] [createPlaylist](https://opensubsonic.netlify.app/docs/endpoints/createplaylist)
- [x] [updatePlaylist](https://opensubsonic.netlify.app/docs/endpoints/updateplaylist)
- [x] [deletePlaylist](https://opensubsonic.netlify.app/docs/endpoints/deleteplaylist)

### Media Retrieval

- [x] [stream](https://opensubsonic.netlify.app/docs/endpoints/stream)
  - [x] raw
  - [x] transcoding (mp3,opus,vorbis,aac), maxBitRate
  - [x] timeOffset
  - [x] estimateContentLength (results in a too large Content-Length value, because it cannot take compression into account)
- [x] [download](https://opensubsonic.netlify.app/docs/endpoints/download)
- [ ] [hls](https://opensubsonic.netlify.app/docs/endpoints/hls)
- [x] [getCoverArt](https://opensubsonic.netlify.app/docs/endpoints/getcoverart)
- [x] [getLyrics](https://opensubsonic.netlify.app/docs/endpoints/getlyrics)
- [x] [getLyricsBySongId](https://opensubsonic.netlify.app/docs/endpoints/getlyricsbysongid)
- [ ] [getAvatar](https://opensubsonic.netlify.app/docs/endpoints/getavatar)

### Media Annotation

- [x] [star](https://opensubsonic.netlify.app/docs/endpoints/star)
- [x] [unstar](https://opensubsonic.netlify.app/docs/endpoints/unstar)
- [x] [setRating](https://opensubsonic.netlify.app/docs/endpoints/setrating)
- [x] [scrobble](https://opensubsonic.netlify.app/docs/endpoints/scrobble)

### Sharing

- [ ] [getShares](https://opensubsonic.netlify.app/docs/endpoints/getshares)
- [ ] [createShare](https://opensubsonic.netlify.app/docs/endpoints/createshare)
- [ ] [updateShare](https://opensubsonic.netlify.app/docs/endpoints/updateshare)
- [ ] [deleteShare](https://opensubsonic.netlify.app/docs/endpoints/deleteshare)

### Podcast

- [ ] [getPodcasts](https://opensubsonic.netlify.app/docs/endpoints/getpodcasts)
- [ ] [getNewestPodcasts](https://opensubsonic.netlify.app/docs/endpoints/getnewestpodcasts)
- [ ] [refreshPodcasts](https://opensubsonic.netlify.app/docs/endpoints/refreshpodcasts)
- [ ] [createPodcastChannel](https://opensubsonic.netlify.app/docs/endpoints/createpodcastchannel)
- [ ] [deletePodcastChannel](https://opensubsonic.netlify.app/docs/endpoints/deletepodcastchannel)
- [ ] [deletePodcastEpisode](https://opensubsonic.netlify.app/docs/endpoints/deletepodcastepisode)
- [ ] [downloadPodcastEpisode](https://opensubsonic.netlify.app/docs/endpoints/downloadpodcastepisode)

### Jukebox

- [ ] [jukeboxControl](https://opensubsonic.netlify.app/docs/endpoints/jukeboxcontrol)

### Internet Radio

**NOTE:** internet radio stations are stored on a *per-user* basis

- [x] [getInternetRadioStations](https://opensubsonic.netlify.app/docs/endpoints/getinternetradiostations)
- [x] [createInternetRadioStation](https://opensubsonic.netlify.app/docs/endpoints/createinternetradiostation)
- [x] [updateInternetRadioStation](https://opensubsonic.netlify.app/docs/endpoints/updateinternetradiostation)
- [x] [deleteInternetRadioStation](https://opensubsonic.netlify.app/docs/endpoints/deleteinternetradiostation)

### User Management

- [ ] [getUser](https://opensubsonic.netlify.app/docs/endpoints/getuser)
- [ ] [getUsers](https://opensubsonic.netlify.app/docs/endpoints/getusers)
- [ ] [createUser](https://opensubsonic.netlify.app/docs/endpoints/createuser)
- [ ] [updateUser](https://opensubsonic.netlify.app/docs/endpoints/updateuser)
- [ ] [deleteUser](https://opensubsonic.netlify.app/docs/endpoints/deleteuser)
- [ ] [changePassword](https://opensubsonic.netlify.app/docs/endpoints/changepassword)

### Bookmarks

- [ ] [getBookmarks](https://opensubsonic.netlify.app/docs/endpoints/getbookmarks)
- [ ] [createBookmark](https://opensubsonic.netlify.app/docs/endpoints/createbookmark)
- [ ] [deleteBookmark](https://opensubsonic.netlify.app/docs/endpoints/deletebookmark)
- [ ] [getPlayQueue](https://opensubsonic.netlify.app/docs/endpoints/getplayqueue)
- [ ] [savePlayQueue](https://opensubsonic.netlify.app/docs/endpoints/saveplayqueue)

### Media Library Scanning

- [x] [getScanStatus](https://opensubsonic.netlify.app/docs/endpoints/getscanstatus)
- [x] [startScan](https://opensubsonic.netlify.app/docs/endpoints/startscan)

### Out of scope
- [search](https://opensubsonic.netlify.app/docs/endpoints/search)
  - *deprecated*, use `search2` or `search3` instead
- [getChatMessages](https://opensubsonic.netlify.app/docs/endpoints/getchatmessages)
- [addChatMessage](https://opensubsonic.netlify.app/docs/endpoints/addchatmessage)
- [getVideos](https://opensubsonic.netlify.app/docs/endpoints/getvideos)
- [getVideoInfo](https://opensubsonic.netlify.app/docs/endpoints/getvideoinfo)
- [getCaptions](https://opensubsonic.netlify.app/docs/endpoints/getcaptions)

## Crossonic

- [x] connectListenBrainz
- [x] getListenBrainzConfig
- [x] setPlaylistCover
- [x] getRecap
- [x] getTopSongsRecap