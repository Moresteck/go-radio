// db
package database

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"

	_ "embed"

	"radio/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
)

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

type Playlist struct {
	Id        int       `json:"id"`
	Name      string    `json:"title"`
	Desc      string    `json:"description"`
	Rank      int       `json:"rank"`
	DebutDate time.Time `json:"debut_date"`
}

type SongData struct {
	SongId      int       `json:"song_id"`
	Authors     string    `json:"authors"`
	Title       string    `json:"title"`
	YTId        string    `json:"youtube_id"`
	Length      int       `json:"length"`
	VoteCount   int       `json:"vote_count"`
	ReleaseDate time.Time `json:"release_date"`
	DebutedAt   time.Time `json:"debuted_at"`
}

type PlaylistEntry struct {
	playlistId int       `json:"playlist_id"`
	songId     int       `json:"song_id"`
	Song       SongData  `json:"song_data"`
	DebutDate  time.Time `json:"debut_date"`
}

type PlaylistEntryArray []PlaylistEntry

func (e PlaylistEntryArray) Len() int {
	return len(e)
}

func (e PlaylistEntryArray) Less(i, j int) bool {
	return e[i].Song.VoteCount > e[j].Song.VoteCount
}

func (e PlaylistEntryArray) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type Vote struct {
	Student    string    `json:"student"`
	VoteType   int       `json:"vote_type"`
	Song       int       `json:"song_id"`
	SubmitDate time.Time `json:"submitted_at"`
}

func GetPlaylistsArrayObject() ([]Playlist, error) {
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

	return playlists, nil
}

func GetPlaylistList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	playlists, err := GetPlaylistsArrayObject()
	if err != nil {
		utils.SendErrorJSON(w, r, "Unknown error")
		log.Printf("DB error: %v\n", err)
		return
	}

	if len(playlists) == 0 {
		utils.SendErrorJSON(w, r, "No playlists found")
	} else {
		j, _ := utils.JSONMarshal(playlists)

		utils.SendJSON(w, r, j)
	}
}

func GetPlaylistInfoObject(playlistid string) *Playlist {
	plarray, _ := GetPlaylistsArrayObject()
	for _, playlist := range plarray {
		id := strconv.Itoa(playlist.Id)
		if id == playlistid {
			return &playlist
		}
	}
	return nil
}

func GetPlaylistObject(playlistid string) ([]PlaylistEntry, error) {

	results, err := db.Query(GetPlaylistTracksQuery, playlistid)
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
		err2 := db.QueryRow(GetTrackQuery, playlistEntry.songId).Scan(&playlistEntry.Song.SongId, &playlistEntry.Song.Authors, &playlistEntry.Song.Title, &playlistEntry.Song.YTId, &playlistEntry.Song.Length, &playlistEntry.Song.ReleaseDate, &playlistEntry.Song.DebutedAt)
		if err2 != nil {
			return playlistEntries, err2
		}

		playlistEntry.Song.VoteCount = len(GetValidVotes(strconv.Itoa(playlistEntry.songId), 1)) - len(GetValidVotes(strconv.Itoa(playlistEntry.songId), 0))
		playlistEntries = append(playlistEntries, playlistEntry)
	}

	return playlistEntries, nil
}

func GetPlaylist(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	playlistid := r.URL.Query().Get("id")
	index, _ := strconv.ParseInt(r.URL.Query().Get("index"), 10, 64)
	index = index*10 - 10

	playlistEntries, err := GetPlaylistObject(playlistid)
	if err != nil {
		utils.SendErrorJSON(w, r, "Unknown error")
		log.Printf("DB error: %v\n", err)
		return
	}
	var toReturn []PlaylistEntry
	// only list 10 entries per request
	currentIndex := int64(-1)
	passed := 0
	for _, obj := range playlistEntries {
		currentIndex++

		if currentIndex < index {
			continue
		}

		if passed >= 10 {
			break
		}

		toReturn = append(toReturn, obj)

		passed++
	}

	if len(toReturn) == 0 {
		utils.SendErrorJSON(w, r, "No playlist with id "+playlistid+" found")
	} else {
		j, _ := utils.JSONMarshal(toReturn)

		utils.SendJSON(w, r, j)
	}
}

func AddPlaylist(title, desc string, rank int) error {
	_, err := db.Exec(AddPlaylistCmd, title, desc, rank)
	return err
}

func DelPlaylist(id string) error {
	_, err := db.Exec(DelPlaylistCmd, id)
	return err
}

func AddTrackToPlaylist(playlistid, songid string) error {
	_, err := db.Exec(AddTrackToPlaylistCmd, playlistid, songid)
	return err
}

func DelTrackFromPlaylist(playlistid, songid string) error {
	_, err := db.Exec(DelTrackFromPlaylistCmd, playlistid, songid)
	return err
}

func AddSong(song SongData) error {
	_, err := db.Exec(AddTrackQuery, song.Authors, song.Title, song.YTId, song.Length, song.ReleaseDate)
	return err
}

func DelSong(songid string) error {
	_, err := db.Exec(DelTrackQuery, songid)
	return err
}

func GetSongObjects() []SongData {
	rows, err := db.Query(GetTracksQuery)
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
		songid := strconv.Itoa(song.SongId)

		song.VoteCount = len(GetValidVotes(songid, 1)) - len(GetValidVotes(songid, 0))

		songs = append(songs, song)
	}
	return songs
}

func GetSongObject(songid string) *SongData {
	song := &SongData{}
	err := db.QueryRow(GetTrackQuery, songid).
		Scan(&song.SongId, &song.Authors, &song.Title, &song.YTId, &song.Length, &song.ReleaseDate, &song.DebutedAt)
	if err != nil {
		log.Printf("DB error (song data): %v\n", err)
		return nil
	} else {
		song.VoteCount = len(GetValidVotes(songid, 1)) - len(GetValidVotes(songid, 0))
		return song
	}
}

func GetSong(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	songid := r.URL.Query().Get("id")
	song := GetSongObject(songid)
	if song != nil {
		j, _ := utils.JSONMarshal(song)

		utils.SendJSON(w, r, j)
	} else {
		utils.SendErrorJSON(w, r, "No song with id "+songid+" found")
	}
}

func GetValidVotes(songid string, votetype int) []Vote {
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

func UpdateVote(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// TODO: verification of group + accesstoken
	userId := r.URL.Query().Get("userId")
	songId := r.URL.Query().Get("songId")
	voteType := r.URL.Query().Get("voteType")
	//accessToken := r.URL.Query().Get("accessToken")

	if song := GetSongObject(songId); song == nil {
		utils.SendErrorJSON(w, r, "Song with given id doesn't exist")
		return
	}

	respond, err := db.Query(VoteQuery, userId, songId)
	if err != nil {
		utils.SendErrorJSON(w, r, "Unknown error")
		log.Printf("DB error (vote update): %v\n", err)
		return
	}

	if respond.Next() {
		_, err2 := db.Exec(UpdateVoteQuery, voteType, userId, songId)
		if err2 != nil {
			utils.SendErrorJSON(w, r, "Unknown error")
			log.Printf("DB error (vote update): %v\n", err2)
			return
		}
	} else {
		_, err2 := db.Exec(AddVoteQuery, userId, voteType, songId)
		if err2 != nil {
			utils.SendErrorJSON(w, r, "Unknown error")
			log.Printf("DB error (vote add): %v\n", err2)
			return
		}
	}
	utils.SendResponseJSON(w, r, "Operation successful")
}

func GetCover(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	songId := r.URL.Query().Get("id")

	coverfile, err := os.Open("music/" + songId + "/cover.jpg")
	if err == nil {
		stat, _ := coverfile.Stat()
		buffer := make([]byte, stat.Size())
		coverfile.Read(buffer)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(buffer)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

type BroadcastTypes struct {
	Playlist PlaylistBroadcastType `json:"playlist"`
	Silence  SilenceBroadcastType  `json:"silence"`
	File     FileBroadcastType     `json:"file"`
}

type PlaylistBroadcastType struct {
	Active     bool   `json:"active"`
	PlaylistId string `json:"playlist_id"`
}

type FileBroadcastType struct {
	Active   bool     `json:"active"`
	Location []string `json:"location_on_disk"`
}

type SilenceBroadcastType struct {
	Active bool `json:"active"`
}

type Range struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Plan struct {
	Range Range          `json:"range"`
	Type  BroadcastTypes `json:"broadcast_type"`
}

type Schedule []Plan

func GetSchedule(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	date := r.URL.Query().Get("date")

	sched := GetScheduleFor(date)

	j, _ := utils.JSONMarshal(sched)

	utils.SendJSON(w, r, j)
}

func GetScheduleFor(date_at string) Schedule {
	var raw string
	var plan []Plan

	//log.Println(date_at)
	err := db.QueryRow(GetPlanCmd, date_at).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Println(err)
		return nil
	}
	rawdecode, _ := base64.StdEncoding.DecodeString(raw)
	//log.Println(string(rawdecode))

	err2 := json.Unmarshal(rawdecode, &plan)
	if err2 != nil {
		log.Println(err2)
		return nil
	}
	return plan
}

func CreateSampleSchedule() {

	start := time.Now()
	end := start.Add(time.Minute * 6)

	plan := []Plan{}
	plan1 := Plan{}
	plan1.Range.Start = start
	plan1.Range.End = end

	start = end.Add(time.Second * 10)
	end = start.Add(time.Minute * 3)

	plan2 := Plan{}
	plan2.Range.Start = start
	plan2.Range.End = end

	pbti := PlaylistBroadcastType{true, "1"}
	plan1.Type.Playlist = pbti
	pbti1 := PlaylistBroadcastType{true, "2"}
	plan2.Type.Playlist = pbti1

	plan = append(plan, plan1)
	plan = append(plan, plan2)

	UpdateSchedule(start.Format("2006-01-02"), plan)

}

func UpdateSchedule(date string, plan []Plan) {
	jnoindent, _ := json.Marshal(plan)

	dbplan := GetScheduleFor(date)

	if dbplan == nil {
		_, err := db.Exec(AddPlanCmd, date, base64.StdEncoding.EncodeToString(jnoindent))
		if err != nil {
			log.Println(err)
		}
	} else {
		_, err := db.Exec(SetPlanCmd, base64.StdEncoding.EncodeToString(jnoindent), date)
		if err != nil {
			log.Println(err)
		}
	}
}

func CreateQueue(playlistid string) []int {
	playlist, err := GetPlaylistObject(playlistid)
	if err != nil {
		return nil
	}
	sort.Sort(PlaylistEntryArray(playlist))

	pos := 0
	var votedList []SongData
	for _, entry := range playlist {
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
	// but check if the same song doesnt occur in the neighbouring two slots

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
