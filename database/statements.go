package database

import _ "embed"

//go:embed queries/getPlaylistList.sql
var GetPlaylistListQuery string

//go:embed queries/getPlaylistTracks.sql
var GetPlaylistTracksQuery string

//go:embed queries/addPlaylist.sql
var AddPlaylistCmd string

//go:embed queries/delPlaylist.sql
var DelPlaylistCmd string

//go:embed queries/addTrackToPlaylist.sql
var AddTrackToPlaylistCmd string

//go:embed queries/delTrackFromPlaylist.sql
var DelTrackFromPlaylistCmd string

//go:embed queries/getTrack.sql
var GetTrackQuery string

//go:embed queries/getTracks.sql
var GetTracksQuery string

//go:embed queries/addTrack.sql
var AddTrackQuery string

//go:embed queries/delTrack.sql
var DelTrackQuery string

//go:embed queries/getVotes.sql
var GetVotesQuery string

//go:embed queries/addVote.sql
var AddVoteQuery string

//go:embed queries/updateVote.sql
var UpdateVoteQuery string

//go:embed queries/queryVote.sql
var VoteQuery string

//go:embed queries/addPlan.sql
var AddPlanCmd string

//go:embed queries/setPlan.sql
var SetPlanCmd string

//go:embed queries/getPlan.sql
var GetPlanCmd string

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
