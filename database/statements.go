package database

import _ "embed"

//go:embed queries/getPlaylistList.sql
var GetPlaylistListQuery string

//go:embed queries/getPlaylistSongs.sql
var GetPlaylistSongsQuery string

//go:embed queries/addPlaylist.sql
var AddPlaylistCmd string

//go:embed queries/delPlaylist.sql
var DelPlaylistCmd string

//go:embed queries/addSongToPlaylist.sql
var AddSongToPlaylistCmd string

//go:embed queries/delSongFromPlaylist.sql
var DelSongFromPlaylistCmd string

//go:embed queries/getSong.sql
var GetSongQuery string

//go:embed queries/getSongs.sql
var GetSongsQuery string

//go:embed queries/addSong.sql
var AddSongCmd string

//go:embed queries/delSong.sql
var DelSongCmd string

//go:embed queries/getVotes.sql
var GetVotesQuery string

//go:embed queries/addVote.sql
var AddVoteCmd string

//go:embed queries/updateVote.sql
var UpdateVoteCmd string

//go:embed queries/queryVote.sql
var VoteQuery string

//go:embed queries/addSchedule.sql
var AddScheduleCmd string

//go:embed queries/setSchedule.sql
var SetScheduleCmd string

//go:embed queries/getSchedule.sql
var GetScheduleQuery string

//go:embed dbschemas/playlists.sql
var SchemaPlaylists string

//go:embed dbschemas/playlist_entries.sql
var SchemaPlaylistEntries string

//go:embed dbschemas/songs.sql
var SchemaSongs string

//go:embed dbschemas/votes.sql
var SchemaVotes string

//go:embed dbschemas/schedule.sql
var SchemaSchedule string
