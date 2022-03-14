package playback

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"

	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"radio/database"
	"radio/fading"
)

var Inited = false
var curPlayList int
var lastPlaylist int
var lastIndex = -1

// [playlistId][queueIndex] = songid
var GeneratedQueues = map[int][]int{}

func InitSpeaker() {
	speaker.Init(44100, int(time.Duration(65536)))
}

func Init() {
	if Inited {
		speaker.Close()
	}
	InitSpeaker()
	playlists, err := database.GetPlaylistsArray()
	if err == nil {
		for _, playlist := range playlists {
			GeneratedQueues[playlist.Id] = database.CreateQueue(strconv.Itoa(playlist.Id))
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
	songids := GeneratedQueues[id]

	// reset the contents in case if another playlist was played before
	streamers = []beep.StreamSeeker{}
	queue = []*database.SongData{}

	var format beep.Format
	startpoint := 0
	if fading.CurFader != nil {
		startpoint = fading.CurFader.Id
	}

	for i := startpoint; i < len(songids); i++ {
		el := songids[i]

		idstr := strconv.Itoa(el)
		song, form := GetSongFormat(idstr)

		if song == nil {
			continue
		}
		dbsong := database.GetSongData(idstr)
		if i == 0 {
			format = form
		}

		streamers = append(streamers, song)
		queue = append(queue, dbsong)
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

func PlaySong(songid string) {
	s := GetSong(songid)
	if s != nil {
		log.Println("Playing " + songid)
		song := database.GetSongData(songid)
		log.Println(song.Authors + " - " + song.Title + " (" + song.ReleaseDate.String() + ")")

		speaker.Play(s)
	}

}

func PlaSongs(songids []string) {
	for _, el := range songids {
		s := GetSong(el)
		if s != nil {
			log.Println("Playing " + el)

			song := database.GetSongData(el)
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
func GetSong(songid string) beep.StreamSeeker {
	buffer, _ := GetSongFormat(songid)
	return buffer
}

func GetSongFormat(songid string) (beep.StreamSeekCloser, beep.Format) {
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

	startpoint := 0
	if fading.CurFader != nil {
		startpoint = fading.CurFader.Id
	}

	for i := startpoint; i < len(files); i++ {
		el := files[i]

		song, form := GetFileStreamer(el)

		if song == nil {
			continue
		}
		if i == 0 {
			format = *form
		}

		streamers = append(streamers, song)
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

var discardCurSchedule = make(map[int]bool)

func StartSchedule(schedule database.Schedule) {
	log.Println("Starting schedule")

	for index, plan := range schedule {

		ticker := time.NewTicker(time.Second)

		go func(plan1 database.PlanBlock, planid int) {
			discardCurSchedule[planid] = false
			phaseout := false
			wasrun := false

			// don't run if it's past the plan's time
			// otherwise conflicts will occur
			if time.Now().After(plan1.Range.End) {
				return
			}

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
								fading.CurFader = nil
								PlayFiles(plan1.Type.File.Location)
								wasrun = true
							}

							if fading.CurFader != nil && lastIndex != fading.CurFader.Id {
								lastIndex = fading.CurFader.Id
								song := fileQueue[lastIndex]
								log.Println("Now playing: " + song)
							}
						} else if plan1.Type.Playlist.Active {
							plid, _ := strconv.ParseInt(plan1.Type.Playlist.PlaylistId, 10, 64)
							pid := int(plid)

							if curPlayList != pid {
								log.Println("Start playback of playlist " + plan1.Type.Playlist.PlaylistId)

								// resume playback in the new planblock
								if lastPlaylist != pid {
									fading.CurFader = nil
								}

								curPlayList = pid
								PlayPlaylist(pid)

							} else if !phaseout && curPlayList == pid &&
								now.After(plan1.Range.End.Add(-time.Second*10)) {

								phaseout = true

								if fading.CurFader != nil {

									// Start fading out
									// This manner pretends that the stream needs to start being faded
									fading.CurFader.AudioLength = float64(0)
								}
							}

							if curPlayList == pid && fading.CurFader != nil && lastIndex != fading.CurFader.Id && len(queue) > fading.CurFader.Id {
								lastIndex = fading.CurFader.Id
								song := queue[lastIndex]
								log.Println(song.Authors + " - " + song.Title + " (" + song.ReleaseDate.Format("2006-01-02") + ")")
							}
						}
					} else if now.After(plan1.Range.End) {

						if plan1.Type.File.Active {

							// Mute before pausing, otherwise it will stutter for half a second
							if curVolume != nil {
								curVolume.Volume = -1
								curVolume.Base *= 10
							}

							// Pause playback before playing the next planblock
							if curCtrl != nil {
								curCtrl.Paused = true
							}

							lastIndex = -1
							fading.CurFader = nil
							fading.Release()
							speaker.Clear()
							ticker.Stop()
							break
						} else if plan1.Type.Playlist.Active {
							plid, _ := strconv.ParseInt(plan1.Type.Playlist.PlaylistId, 10, 64)

							if curPlayList == int(plid) {
								log.Println("End playback of playlist " + plan1.Type.Playlist.PlaylistId)

								// Mute before pausing, otherwise it will stutter for half a second
								if curVolume != nil {
									curVolume.Volume = -1
									curVolume.Base *= 10
								}

								// Pause playback before playing the next planblock
								if curCtrl != nil {
									curCtrl.Paused = true
								}

								curPlayList = -1
								lastIndex = -1
								lastPlaylist = int(plid)
								fading.Release()
								speaker.Clear()
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
		Init()
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
