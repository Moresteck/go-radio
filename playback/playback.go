package playback

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/julienschmidt/httprouter"

	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"net/http"

	"radio/database"
	"radio/fading"
)

var Inited = false
var curPlayList int
var lastPlaylist int
var lastIndex = -1

// [playlistId][queueIndex] = songid
var generatedQueues = map[int][]int{}

func InitSpeaker() {
	speaker.Init(44100, int(time.Duration(65536)))
}

func Init() {
	InitSpeaker()
	playlists, err := database.GetPlaylistsArrayObject()
	if err == nil {
		for _, playlist := range playlists {
			generatedQueues[playlist.Id] = database.CreateQueue(strconv.Itoa(playlist.Id))
		}
		Inited = true
	}
}

var streamers []beep.StreamSeeker
var queue []*database.SongData
var fileQueue []string

var curStreamer beep.Streamer
var curCtrl *beep.Ctrl
var curVolume *effects.Volume

func PlayPlaylist(id int) {
	songids := generatedQueues[id]

	// reset the contents in case if another playlist was played before
	streamers = []beep.StreamSeeker{}
	queue = []*database.SongData{}

	var format beep.Format

	for i := fading.CurIndex; i < len(songids); i++ {
		el := songids[i]

		idstr := strconv.Itoa(el)
		track, form := GetTrackFormat(idstr)

		if track == nil {
			continue
		}
		dbtrack := database.GetSongObject(idstr)
		if i == 0 {
			format = form
		}

		streamers = append(streamers, track)
		queue = append(queue, dbtrack)
	}
	opts := fading.Options{
		TimeSpan: time.Duration(5) * time.Second,
		Volume:   1,
	}
	curStreamer = fading.CrossfadeStream(format, &opts, streamers...)
	curCtrl = &beep.Ctrl{Streamer: curStreamer, Paused: false}
	curVolume = &effects.Volume{
		Streamer: curCtrl,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	// speaker.Play(final)
	speaker.Play(curVolume)
}

func PlayTrack(songid string) {
	s := GetTrack(songid)
	if s != nil {
		log.Println("Playing " + songid)
		song := database.GetSongObject(songid)
		log.Println(song.Authors + " - " + song.Title + " (" + song.ReleaseDate.String() + ")")

		speaker.Play(s)
	}

}

func PlayTracks(songids []string) {
	for _, el := range songids {
		s := GetTrack(el)
		if s != nil {
			log.Println("Playing " + el)

			song := database.GetSongObject(el)
			log.Println(song.Authors + " - " + song.Title + " (" + song.ReleaseDate.String() + ")")
			done := make(chan bool)
			speaker.Play(beep.Seq(s, beep.Callback(func() {
				done <- true
			})))

			<-done
		}

	}
	log.Println("End playback")
}
func GetTrack(songid string) beep.StreamSeeker {
	buffer, _ := GetTrackFormat(songid)
	return buffer
}

func GetTrackFormat(songid string) (beep.StreamSeekCloser, beep.Format) {
	if _, err := os.Stat("music/" + songid + "/audio.wav"); err == nil {
		f, err := os.Open("music/" + songid + "/audio.wav")
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		streamer, format, err := wav.Decode(f)
		if err != nil {
			log.Println(err)
		}

		return streamer, format
	} else if _, err := os.Stat("music/" + songid + "/audio.mp3"); err == nil {
		f, err := os.Open("music/" + songid + "/audio.mp3")
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		streamer, format, err := mp3.Decode(f)
		if err != nil {
			log.Println(err)
		}

		return streamer, format
	} else if _, err := os.Stat("music/" + songid + "/audio.flac"); err == nil {
		f, err := os.Open("music/" + songid + "/audio.flac")
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		streamer, format, err := flac.Decode(f)
		if err != nil {
			log.Println(err)
		}

		return streamer, format
	} else {
		log.Println("No song with id " + songid)
		return nil, beep.Format{}
	}

}

func PlayFiles(files []string) {

	// reset the contents in case if another playlist was played before
	streamers = []beep.StreamSeeker{}
	fileQueue = []string{}

	var format beep.Format

	for i := fading.CurIndex; i < len(files); i++ {
		el := files[i]

		track, form := GetFileStreamer(el)

		if track == nil {
			continue
		}
		if i == 0 {
			format = *form
		}

		streamers = append(streamers, track)
		fileQueue = append(fileQueue, el)
	}
	opts := fading.Options{
		TimeSpan: time.Duration(5) * time.Second,
		Volume:   1,
	}
	curStreamer = fading.CrossfadeStream(format, &opts, streamers...)
	curCtrl = &beep.Ctrl{Streamer: curStreamer, Paused: false}
	curVolume = &effects.Volume{
		Streamer: curCtrl,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	// speaker.Play(final)
	speaker.Play(curVolume)
}

func GetFileStreamer(loc string) (beep.StreamSeeker, *beep.Format) {
	f, err := os.Open(loc)
	if err != nil {
		log.Println("Can't play '" + loc + "' - file doesn't exist or is inaccessible")
		log.Println(err)
		return nil, nil
	}

	if strings.HasSuffix(loc, ".flac") {
		streamer, format, err1 := flac.Decode(f)
		if err1 != nil {
			log.Println(err1)
		}

		return streamer, &format
	} else if strings.HasSuffix(loc, ".mp3") {
		streamer, format, err1 := mp3.Decode(f)
		if err1 != nil {
			log.Println(err1)
		}

		return streamer, &format
	} else if strings.HasSuffix(loc, ".wav") {
		streamer, format, err1 := wav.Decode(f)
		if err1 != nil {
			log.Println(err1)
		}

		return streamer, &format
	} else {
		return nil, nil
	}
}

func Play(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	if Inited == false {
		Init()
	}

	songs := []string{"2", "1"}
	PlayTracks(songs)
}

var discardCurSchedule = make(map[int]bool)

func StartSchedule(schedule database.Schedule) {
	log.Println("Starting schedule")

	for index, plan := range schedule {

		ticker := time.NewTicker(time.Second)

		go func(plan1 database.Plan, planid int) {
			discardCurSchedule[planid] = false
			phaseout := false
			wasrun := false
			for {
				select {
				case <-ticker.C:
					//log.Println("tick playlist " + plan1.Type.Playlist.PlaylistId)
					if discardCurSchedule[planid] {
						if curCtrl != nil {
							curCtrl.Paused = true
						}

						curPlayList = -1
						lastPlaylist = -1
						lastIndex = -1
						fading.Release()
						ticker.Stop()
						break
					}
					now := time.Now()
					if !now.Before(plan1.Range.Start) && !now.After(plan1.Range.End) {
						if plan1.Type.File.Active {
							if !wasrun {
								lastPlaylist = -1
								fading.CurIndex = 0
								PlayFiles(plan1.Type.File.Location)
								wasrun = true
							}

							if lastIndex != fading.CurIndex {
								lastIndex = fading.CurIndex
								song := fileQueue[fading.CurIndex]
								log.Println("Now playing: " + song)
							}
						} else if plan1.Type.Playlist.Active {
							plid, _ := strconv.ParseInt(plan1.Type.Playlist.PlaylistId, 10, 64)
							pid := int(plid)

							if curPlayList != pid {
								log.Println("Start playback of playlist " + plan1.Type.Playlist.PlaylistId)

								// resume playback in the new planblock
								if lastPlaylist != pid {
									fading.CurIndex = 0
								}

								curPlayList = pid
								PlayPlaylist(pid)

							} else if !phaseout && curPlayList == pid &&
								now.After(plan1.Range.End.Add(-time.Second*10)) {

								phaseout = true

								if fading.CurIndex > -1 {

									go func(plan2 database.Plan, planid1 int) {
										//lowvolumeticker := time.NewTicker(time.Second / 10)

										for {
											if discardCurSchedule[planid1] {
												break
											}
											now1 := time.Now()

											if now1.After(plan2.Range.End) {
												//lowvolumeticker.Stop()
												break
											} else if curVolume != nil {
												//volume := volumes[curIndex]

												speaker.Lock()
												curVolume.Volume -= 0.01
												curVolume.Base *= 2
												speaker.Unlock()
												time.Sleep(time.Millisecond * 66)
											}
										}
									}(plan1, planid)

								}
							}

							if curPlayList == pid && lastIndex != fading.CurIndex && len(queue) > fading.CurIndex {
								lastIndex = fading.CurIndex
								song := queue[fading.CurIndex]
								log.Println(song.Authors + " - " + song.Title + " (" + song.ReleaseDate.Format("2006-01-02") + ")")
							}
						}
					} else if now.After(plan1.Range.End) {

						if plan1.Type.File.Active {
							// Pause playback before playing the next planblock
							if curCtrl != nil {
								curCtrl.Paused = true
							}

							lastIndex = -1
							fading.CurIndex = 0
							fading.Release()
							ticker.Stop()
							break
						} else if plan1.Type.Playlist.Active {
							plid, _ := strconv.ParseInt(plan1.Type.Playlist.PlaylistId, 10, 64)

							if curPlayList == int(plid) {
								log.Println("End playback of playlist " + plan1.Type.Playlist.PlaylistId)

								// Pause playback before playing the next planblock
								if curCtrl != nil {
									curCtrl.Paused = true
								}

								curPlayList = -1
								lastIndex = -1
								lastPlaylist = int(plid)
								fading.Release()
								ticker.Stop()
								break
							}

						}
					}
				}
			}
		}(plan, index)
	}

}

func ScheduleChanged(at_date string) {
	now := time.Now()
	str := now.Format("2006-01-02")
	if str == at_date {
		for index, _ := range discardCurSchedule {
			discardCurSchedule[index] = true
		}
		log.Println("WAIT")
		time.Sleep(time.Second * 2)
		PlayTodaySchedule()
	}
}

func PlayTodaySchedule() {
	now := time.Now()
	schedule := database.GetScheduleFor(now.Format("2006-01-02"))
	if schedule == nil {
		log.Println("No schedule planned for today!")
	} else {
		StartSchedule(schedule)
	}
}
