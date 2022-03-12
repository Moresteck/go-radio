package database

import (
	"log"
	"net/http"
	"os"
	"radio/utils"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func HTTPGetPlaylistIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	playlists, err := GetPlaylistsArray()
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

func HTTPGetPlaylist(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	playlistid := r.URL.Query().Get("id")
	index, _ := strconv.ParseInt(r.URL.Query().Get("index"), 10, 64)
	index = index*10 - 10

	playlistEntries, err := GetPlaylistEntries(playlistid)
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

func HTTPGetSong(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	songid := r.URL.Query().Get("id")
	song := GetSongData(songid)
	if song != nil {
		j, _ := utils.JSONMarshal(song)

		utils.SendJSON(w, r, j)
	} else {
		utils.SendErrorJSON(w, r, "No song with id "+songid+" found")
	}
}

func HTTPUpdateVote(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// TODO: verification of group + accesstoken
	userId := r.URL.Query().Get("userId")
	songId := r.URL.Query().Get("songId")
	voteType := r.URL.Query().Get("voteType")
	//accessToken := r.URL.Query().Get("accessToken")

	if song := GetSongData(songId); song == nil {
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
		_, err2 := db.Exec(UpdateVoteCmd, voteType, userId, songId)
		if err2 != nil {
			utils.SendErrorJSON(w, r, "Unknown error")
			log.Printf("DB error (vote update): %v\n", err2)
			return
		}
	} else {
		_, err2 := db.Exec(AddVoteCmd, userId, voteType, songId)
		if err2 != nil {
			utils.SendErrorJSON(w, r, "Unknown error")
			log.Printf("DB error (vote add): %v\n", err2)
			return
		}
	}
	utils.SendResponseJSON(w, r, "Operation successful")
}

func HTTPGetCover(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

func HTTPGetSchedule(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	date := r.URL.Query().Get("date")

	sched := GetScheduleFor(date)

	j, _ := utils.JSONMarshal(sched)

	utils.SendJSON(w, r, j)
}
