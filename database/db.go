// db
package database

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"

	"math/rand"
	"sort"
	"strconv"
	"time"

	_ "embed"

	_ "github.com/go-sql-driver/mysql"
)

// db connection
var db *sql.DB

func Init() {
	log.Println("Initializing MySQL")

	var err error
	db, err = sql.Open("mysql", "root:kopytko@/radio?parseTime=true")
	if err != nil {
		log.Fatalf("Couldn't open database connection: %v\n", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Couldn't ping database connection: %v\n", err)
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)

	// populate defaults if they don't exist already
	if _, err := db.Exec(SchemaPlaylists); err != nil {
		log.Fatalf("Couldn't prepare playlist table: %v\n", err)
	}

	if _, err := db.Exec(SchemaPlaylistEntries); err != nil {
		log.Fatalf("Couldn't prepare playlist entries table: %v\n", err)
	}

	if _, err := db.Exec(SchemaSongs); err != nil {
		log.Fatalf("Couldn't prepare songs table: %v\n", err)
	}

	if _, err := db.Exec(SchemaVotes); err != nil {
		log.Fatalf("Couldn't prepare votes table: %v\n", err)
	}

	if _, err := db.Exec(SchemaSchedule); err != nil {
		log.Fatalf("Couldn't prepare schedule table: %v\n", err)
	}
}

// playlist object data
type Playlist struct {
	Id   int    `json:"id"`
	Name string `json:"title"`
	Desc string `json:"description"`
	// the higher the rank, the earlier the playlist will show on playlist index
	Rank      int       `json:"rank"`
	DebutDate time.Time `json:"debut_date"`
}

// PlaylistArray to sort by rank
type PlaylistArray []Playlist

func (e PlaylistArray) Len() int {
	return len(e)
}

func (e PlaylistArray) Less(i, j int) bool {
	return e[i].Rank > e[j].Rank
}

func (e PlaylistArray) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// song object as seen in db
type SongData struct {
	SongId      int       `json:"song_id"`
	Authors     string    `json:"authors"`
	Title       string    `json:"title"`
	YTId        string    `json:"youtube_id"`
	Length      int       `json:"length"`
	ReleaseDate time.Time `json:"release_date"`
	DebutedAt   time.Time `json:"debuted_at"`
}

// delegated to a method, because vote count is dynamic
func (song *SongData) VoteCount() int {
	// substract negative votes from positive ones to get the total value
	// 1 - positive; 0 - negative
	return len(GetValidVotesForSong(strconv.Itoa(song.SongId), 1)) - len(GetValidVotesForSong(strconv.Itoa(song.SongId), 0))
}

// playlist entry from db
type PlaylistEntry struct {
	playlistId int       `json:"playlist_id"`
	songId     int       `json:"song_id"`
	DebutDate  time.Time `json:"debut_date"`

	// this isn't stored in db; it's for easy access
	Song SongData `json:"song_data"`
}

type PlaylistEntryArray []PlaylistEntry

func (e PlaylistEntryArray) Len() int {
	return len(e)
}

func (e PlaylistEntryArray) Less(i, j int) bool {
	return e[i].Song.VoteCount() > e[j].Song.VoteCount()
}

func (e PlaylistEntryArray) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// vote object from db
type Vote struct {
	Student string `json:"student"`
	// 0 - negative; 1 - positive
	VoteType   int       `json:"vote_type"`
	Song       int       `json:"song_id"`
	SubmitDate time.Time `json:"submitted_at"`
}

func GetPlaylistsArray() ([]Playlist, error) {
	results, err := db.Query(GetPlaylistListQuery)
	if err != nil {
		return nil, err
	}

	var playlists []Playlist
	for results.Next() {
		var playlist Playlist

		err = results.Scan(&playlist.Id, &playlist.Name, &playlist.Desc, &playlist.Rank, &playlist.DebutDate)
		if err != nil {
			return playlists, err
		}
		playlists = append(playlists, playlist)
	}

	sort.Sort(PlaylistArray(playlists))

	return playlists, nil
}

func GetPlaylistData(playlistid string) *Playlist {
	plarray, _ := GetPlaylistsArray()
	for _, playlist := range plarray {
		id := strconv.Itoa(playlist.Id)
		if id == playlistid {
			return &playlist
		}
	}
	return nil
}

func GetPlaylistEntries(playlistid string) ([]PlaylistEntry, error) {

	results, err := db.Query(GetPlaylistSongsQuery, playlistid)
	if err != nil {
		return nil, err
	}

	var playlistEntries []PlaylistEntry
	for results.Next() {
		var playlistEntry PlaylistEntry

		err = results.Scan(&playlistEntry.playlistId, &playlistEntry.songId, &playlistEntry.DebutDate)
		if err != nil {
			return playlistEntries, err
		}
		err2 := db.QueryRow(GetSongQuery, playlistEntry.songId).Scan(&playlistEntry.Song.SongId, &playlistEntry.Song.Authors, &playlistEntry.Song.Title, &playlistEntry.Song.YTId, &playlistEntry.Song.Length, &playlistEntry.Song.ReleaseDate, &playlistEntry.Song.DebutedAt)
		if err2 != nil {
			return playlistEntries, err2
		}

		playlistEntries = append(playlistEntries, playlistEntry)
	}

	return playlistEntries, nil
}

func AddPlaylist(title, desc string, rank int) error {
	_, err := db.Exec(AddPlaylistCmd, title, desc, rank)
	return err
}

func DelPlaylist(id string) error {
	_, err := db.Exec(DelPlaylistCmd, id)
	return err
}

func AddSongToPlaylist(playlistid, songid string) error {
	_, err := db.Exec(AddSongToPlaylistCmd, playlistid, songid)
	return err
}

func DelSongFromPlaylist(playlistid, songid string) error {
	_, err := db.Exec(DelSongFromPlaylistCmd, playlistid, songid)
	return err
}

func AddSong(song SongData) error {
	_, err := db.Exec(AddSongCmd, song.Authors, song.Title, song.YTId, song.Length, song.ReleaseDate)
	return err
}

func DelSong(songid string) error {
	_, err := db.Exec(DelSongCmd, songid)
	return err
}

func GetSongArray() []SongData {
	rows, err := db.Query(GetSongsQuery)
	if err != nil {
		return []SongData{}
	}

	var songs []SongData
	for rows.Next() {
		var song SongData

		err := rows.Scan(&song.SongId, &song.Authors, &song.Title, &song.YTId, &song.Length, &song.ReleaseDate, &song.DebutedAt)
		if err != nil {
			return []SongData{}
		}

		songs = append(songs, song)
	}
	return songs
}

func GetSongData(songid string) *SongData {
	song := &SongData{}
	err := db.QueryRow(GetSongQuery, songid).
		Scan(&song.SongId, &song.Authors, &song.Title, &song.YTId, &song.Length, &song.ReleaseDate, &song.DebutedAt)
	if err != nil {
		log.Printf("DB error (song data): %v\n", err)
		return nil
	} else {
		return song
	}
}

func GetValidVotesForSong(songid string, votetype int) []Vote {
	results, err := db.Query(GetVotesQuery, songid, votetype)
	if err != nil {
		return []Vote{}
	}

	var votes []Vote
	for results.Next() {
		var vote Vote

		err = results.Scan(&vote.Student, &vote.VoteType, &vote.Song, &vote.SubmitDate)
		if err != nil {
			return []Vote{}
		}
		votes = append(votes, vote)
	}

	return votes
}

// this tells the broadcast type of a planblock
// only ONE type from this struct can be Active=true
type BroadcastTypes struct {
	Playlist PlaylistBroadcastType `json:"playlist"`
	Silence  SilenceBroadcastType  `json:"silence"`
	File     FileBroadcastType     `json:"file"`
}

// abstract type inherited by its implementations
type BroadcastType struct {
	Active bool `json:"active"`
}

type PlaylistBroadcastType struct {
	BroadcastType
	PlaylistId string `json:"playlist_id"`
}

type FileBroadcastType struct {
	BroadcastType
	// values in array must point to existing wav/mp3/flac files
	Location []string `json:"location_on_disk"`
}

type SilenceBroadcastType struct {
	BroadcastType
}

type Range struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type PlanBlock struct {
	Range Range          `json:"range"`
	Type  BroadcastTypes `json:"broadcast_type"`
}

// a schedule consists of planblocks
type Schedule []PlanBlock

func GetScheduleFor(date_at string) Schedule {
	var raw string
	var schedule Schedule

	//log.Println(date_at)
	err := db.QueryRow(GetScheduleQuery, date_at).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Println(err)
		return nil
	}
	rawdecode, _ := base64.StdEncoding.DecodeString(raw)
	//log.Println(string(rawdecode))

	err2 := json.Unmarshal(rawdecode, &schedule)
	if err2 != nil {
		log.Println(err2)
		return nil
	}
	return schedule
}

func CreateSampleSchedule() {

	start := time.Now()
	end := start.Add(time.Second * 30)

	schedule := Schedule{}
	plan1 := PlanBlock{}
	plan1.Range.Start = start
	plan1.Range.End = end

	start = end.Add(time.Second * 10)
	end = start.Add(time.Minute * 3)

	plan2 := PlanBlock{}
	plan2.Range.Start = start
	plan2.Range.End = end

	pbti := PlaylistBroadcastType{BroadcastType: BroadcastType{Active: true}, PlaylistId: "1"}
	plan1.Type.Playlist = pbti
	pbti1 := PlaylistBroadcastType{BroadcastType: BroadcastType{Active: true}, PlaylistId: "2"}
	plan2.Type.Playlist = pbti1

	schedule = append(schedule, plan1)
	schedule = append(schedule, plan2)

	UpdateSchedule(start.Format("2006-01-02"), schedule)

}

func UpdateSchedule(date string, schedule Schedule) {
	jnoindent, _ := json.Marshal(schedule)

	dbschedule := GetScheduleFor(date)

	if dbschedule == nil {
		_, err := db.Exec(AddScheduleCmd, date, base64.StdEncoding.EncodeToString(jnoindent))
		if err != nil {
			log.Println(err)
		}
	} else {
		_, err := db.Exec(SetScheduleCmd, base64.StdEncoding.EncodeToString(jnoindent), date)
		if err != nil {
			log.Println(err)
		}
	}
}

// returns an array of songids
func CreateQueue(playlistid string) []int {
	entries, err := GetPlaylistEntries(playlistid)
	if err != nil {
		return nil
	}
	sort.Sort(PlaylistEntryArray(entries))

	pos := 0
	var votedList []SongData
	for _, entry := range entries {
		pos++

		var multiply int
		song := entry.Song
		if pos >= 1 || pos <= 3 {
			multiply = 5
		} else if pos >= 4 || pos <= 9 {
			multiply = 4
		} else if pos >= 10 || pos <= 16 {
			multiply = 3
		} else if pos >= 17 || pos <= 24 {
			multiply = 2
		} else {
			multiply = 1
		}

		for i := 0; i < multiply; i++ {
			votedList = append(votedList, song)
		}
	}

	// get random song from the pile
	// then append it to the final queue
	// but check if the same song doesnt occur in the three last slots

	var queue []int
	rand.Seed(time.Now().UnixNano())
	pos = 0
	for i := 0; i < 50; i++ {
		if len(votedList) == 0 {
			continue
		}
		song := votedList[rand.Intn(len(votedList))]

		if len(queue) >= 1 {
			if queue[pos-1] == song.SongId {
				continue
			}
			if len(queue) >= 2 {
				if queue[pos-2] == song.SongId {
					continue
				}
			}
			if len(queue) >= 3 {
				if queue[pos-3] == song.SongId {
					continue
				}
			}
		}

		queue = append(queue, song.SongId)

		pos++
	}

	return queue
}
