package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"radio/database"
	"radio/playback"
	"radio/session"
	"radio/utils"

	"github.com/julienschmidt/httprouter"
)

var addr = flag.String("addr", ":2137", "TCP address to listen on")
var debugMode = flag.Bool("debug", false, "Enable debug mode")

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	database.Init()

	log.Println("Hello World!")

	router := httprouter.New()

	router.MethodNotAllowed = &MethodNotAllowedHandler{}
	router.NotFound = &NotFoundHandler{}
	router.RedirectTrailingSlash = true

	router.GET("/updatevote", database.UpdateVote)
	router.GET("/play", playback.Play)
	router.GET("/getplaylists", database.GetPlaylistList)
	router.GET("/getplaylist", database.GetPlaylist)
	router.GET("/getsong", database.GetSong)
	router.GET("/getcover", database.GetCover)
	router.GET("/getplan", database.GetSchedule)

	database.CreateSampleSchedule()

	// Init the speaker and random queue of playlists
	playback.Init()

	// Load schedule for today
	playback.PlayTodaySchedule()

	go consoleInput()

	log.Printf("Listening at %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, session.New(session.NewProtect(router), *debugMode)))
}

type MethodNotAllowedHandler struct {
}

type NotFoundHandler struct {
}

func (h *MethodNotAllowedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	utils.SendHTTP(w, r, "method not allowed", "bad request")
}

func (h *NotFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	utils.SendHTTP(w, r, "not found", "bad request")
}

var reader *bufio.Reader

func consoleInput() {
	for {
		reader = bufio.NewReader(os.Stdin)
		// ReadString will block until the delimiter is entered
		input, err := reader.ReadString('\n')
		if cmdHandleErr(err) {
			continue
		}

		input = strings.TrimSuffix(input, "\r\n")

		// TODO query (by matching title, authors)
		// get queue (by playlist id)
		args := strings.Split(input, " ")
		if args[0] == "playlist" {
			if len(args) < 2 {
				printHelp(args[0])
				continue
			}

			if args[1] == "get" {
				if len(args) < 3 {
					printHelp(args[0])
					continue
				}

				plid := args[2]

				plistobj := database.GetPlaylistInfoObject(plid)
				plist, err := database.GetPlaylistObject(plid)
				if cmdHandleErr(err) {
					break
				}
				index := 0
				pagestr := "1"

				if len(args) == 4 {
					pagestr = args[3]
					page, err := strconv.ParseInt(args[3], 10, 64)
					if cmdHandleErr(err) {
						break
					}
					index = int(page)*10 - 10
				}

				var maxpage int
				maxpage = int((len(plist)-1)/10) + 1
				fmt.Println("=== Playlist " + plid + " (" + plistobj.Name + ") ===")
				fmt.Println("Size: " + strconv.Itoa(len(plist)))
				fmt.Println("Page: " + pagestr + "/" + strconv.Itoa(maxpage))
				fmt.Println()
				passed := 0
				for currentIndex, entry := range plist {
					song := entry.Song

					if currentIndex < index {
						continue
					}

					if passed >= 10 {
						break
					}

					printSong(song)
					passed++
				}
				fmt.Println("=== Playlist end ===")
			} else if args[1] == "addsong" {
				if len(args) < 4 {
					printHelp(args[0])
					continue
				}

				err = database.AddTrackToPlaylist(args[2], args[3])
				if !cmdHandleErr(err) {
					log.Println("Song " + args[3] + " was successfully added to playlist " + args[2] + "!")
				}
			} else if args[1] == "remsong" {
				if len(args) < 4 {
					printHelp(args[0])
					continue
				}

				err = database.DelTrackFromPlaylist(args[2], args[3])
				if !cmdHandleErr(err) {
					log.Println("Song " + args[3] + " was successfully removed from playlist " + args[2] + "!")
				}
			} else if args[1] == "delete" {
				if len(args) < 3 {
					printHelp(args[0])
					continue
				}

				list := database.GetPlaylistInfoObject(args[2])

				fmt.Println("=Are you sure you want to delete playlist " + args[2] + " [" + list.Name + "? (y/n)")
				yn, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				yn = strings.TrimSuffix(yn, "\r\n")

				if yn == "y" {
					err = database.DelPlaylist(args[2])
					if !cmdHandleErr(err) {
						log.Println("Playlist deleted successfully!")
					}
				} else {
					log.Println("Playlist deletion canceled.")
				}
			} else if args[1] == "list" {
				list, err := database.GetPlaylistsArrayObject()
				if cmdHandleErr(err) {
					break
				}
				for _, pl := range list {
					printPlaylist(pl)
				}
			} else if args[1] == "create" {
				fmt.Println("=Title:")
				title, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				title = strings.TrimSuffix(title, "\r\n")

				fmt.Println("=Description:")
				desc, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				desc = strings.TrimSuffix(desc, "\r\n")

				fmt.Println("=Ranking (the bigger number the higher this playlist will show on the root playlist index):")
				var rank int
				_, err = fmt.Scanf("%d", &rank)
				if cmdHandleErr(err) {
					break
				}

				err = database.AddPlaylist(title, desc, rank)
				if !cmdHandleErr(err) {
					log.Println("Playlist created successfully!")
				}
			}
		} else if args[0] == "song" {
			if len(args) < 2 {
				printHelp(args[0])
				continue
			}

			if args[1] == "list" {
				songs := database.GetSongObjects()
				index := 0

				if len(args) == 3 {
					page, err := strconv.ParseInt(args[2], 10, 64)
					if cmdHandleErr(err) {
						break
					}
					index = int(page)*10 - 10
				}

				passed := 0
				for currentIndex, song := range songs {

					if currentIndex < index {
						continue
					}

					if passed >= 10 {
						break
					}

					printSong(song)
					passed++
				}
			} else if args[1] == "add" {
				fmt.Println("=Title:")
				title, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				title = strings.TrimSuffix(title, "\r\n")

				fmt.Println("=Authors:")
				authors, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				authors = strings.TrimSuffix(authors, "\r\n")

				fmt.Println("=YouTube:")
				yt, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				yt = strings.TrimSuffix(yt, "\r\n")

				fmt.Println("=Release date:")
				date, err := reader.ReadString('\n')
				if cmdHandleErr(err) {
					break
				}
				date = strings.TrimSuffix(date, "\r\n")

				fmt.Println("=Length (in seconds):")
				var length int
				_, err = fmt.Scanf("%d", &length)
				if cmdHandleErr(err) {
					break
				}

				resdate, err := time.Parse("2006-01-02", date)
				if cmdHandleErr(err) {
					break
				}

				song := database.SongData{
					Authors:     authors,
					Title:       title,
					Length:      int(length),
					YTId:        yt,
					ReleaseDate: resdate,
				}

				err = database.AddSong(song)
				if !cmdHandleErr(err) {
					log.Println("Song added successfully!")
				}
			} else if args[1] == "delete" {
				if len(args) < 3 {
					printHelp(args[0])
					continue
				}

				err = database.DelSong(args[2])
				if !cmdHandleErr(err) {
					log.Println("Song deleted successfully!")
				}
			}
		} else if args[0] == "schedule" {
			if len(args) < 2 {
				printHelp(args[0])
				continue
			}

			if args[1] == "today" {
				schedule := database.GetScheduleFor(time.Now().Format("2006-01-02"))
				for index, plan := range schedule {
					fmt.Println("Pos " + strconv.Itoa(index))
					printPlan(plan)
				}
				// schedule add YYYY-MM-dd
			} else if args[1] == "change" {
				if len(args) < 3 {
					printHelp(args[0])
					continue
				}
				date := args[2]
				schedule := database.GetScheduleFor(date)
				if schedule == nil {
					log.Println("No schedule was planned for '" + date + "'!")
					continue
				}
				for index, plan := range schedule {
					fmt.Println("Pos " + strconv.Itoa(index))
					printPlan(plan)
				}

				for {
					fmt.Println("=Interact with (pos index | no | new):")
					posstr, err0 := reader.ReadString('\n')
					if cmdHandleErr(err0) {
						break
					}

					posstr = strings.TrimSuffix(posstr, "\r\n")
					if posstr == "no" {
						break
					} else if posstr == "new" {
						timestart, timeend := readTime(date)
						if timestart == nil {
							break
						}

						plan := readBroadcastType(*timestart, *timeend)
						if plan == nil {
							break
						}
						schedule = append(schedule, *plan)
						database.UpdateSchedule(date, schedule)
						playback.ScheduleChanged(date)
						log.Println("Schedule updated successfully!")
						fmt.Println("Pos " + strconv.Itoa(len(schedule)-1))
						printPlan(*plan)
						continue
					}
					pos64, err1 := strconv.ParseInt(posstr, 10, 64)
					if cmdHandleErr(err1) {
						break
					}
					pos := int(pos64)

					if pos >= len(schedule) || pos < 0 {
						log.Println("Pos index nil!!!")
						break
					}

					fmt.Println("= (change | remove)")
					action, err2 := reader.ReadString('\n')
					if cmdHandleErr(err2) {
						break
					}
					action = strings.TrimSuffix(action, "\r\n")
					if action == "change" {
						fmt.Println("=Range start (keep | <HH:mm:ss>)")
						rangestartarg, err3 := reader.ReadString('\n')
						if cmdHandleErr(err3) {
							break
						}
						rangestartarg = strings.TrimSuffix(rangestartarg, "\r\n")

						if rangestartarg != "keep" {
							rangestartarg = date + "T" + rangestartarg + "-00:00"
							rangest, terr1 := time.Parse(time.RFC3339, rangestartarg)
							if cmdHandleErr(terr1) {
								break
							}
							schedule[pos].Range.Start = rangest
						}

						fmt.Println("=Range end (keep | <HH:mm:ss>)")
						rangeendarg, err4 := reader.ReadString('\n')
						if cmdHandleErr(err4) {
							break
						}
						rangeendarg = strings.TrimSuffix(rangeendarg, "\r\n")

						if rangeendarg != "keep" {
							rangeendarg = date + "T" + rangeendarg + "-00:00"
							rangeet, terr1 := time.Parse(time.RFC3339, rangeendarg)
							if cmdHandleErr(terr1) {
								break
							}
							schedule[pos].Range.End = rangeet
						}

						fmt.Println("=Edit type? (yes | no)")
						yn, err5 := reader.ReadString('\n')
						if cmdHandleErr(err5) {
							break
						}
						yn = strings.TrimSuffix(yn, "\r\n")

						if yn == "yes" {
							plan := readBroadcastType(schedule[pos].Range.Start, schedule[pos].Range.End)
							if plan == nil {
								break
							}
							schedule[pos] = *plan
						}
						database.UpdateSchedule(date, schedule)
						playback.ScheduleChanged(date)
						log.Println("Schedule updated successfully!")
					} else if action == "remove" {
						// make it silent
						schedule[pos].Type.Silence.Active = true
						schedule[pos].Type.Playlist.Active = false
						schedule[pos].Type.Playlist.PlaylistId = ""
						schedule[pos].Type.File.Active = false
						schedule[pos].Type.File.Location = []string{}

						database.UpdateSchedule(date, schedule)
						playback.ScheduleChanged(date)
						log.Println("Schedule updated successfully!")
					}
				}
				// schedule set YYYY-MM-dd
			} else if args[1] == "set" {
				if len(args) < 3 {
					printHelp(args[0])
					continue
				}
				date := args[2]
				var plans []database.Plan
				for {
					fmt.Println("== New block? (yes | no)")
					yesno, err0 := reader.ReadString('\n')
					if cmdHandleErr(err0) {
						break
					}
					yesno = strings.Replace(yesno, "\r\n", "", -1)
					if yesno == "no" {
						break
					}

					timestart, timeend := readTime(date)
					if timestart == nil {
						break
					}

					plan := readBroadcastType(*timestart, *timeend)
					if plan == nil {
						break
					}
					plans = append(plans, *plan)
				}
				database.UpdateSchedule(date, plans)
				playback.ScheduleChanged(date)
				log.Println("Schedule for '" + date + "' successfully set!")
			}
		} else {
			fmt.Println("Unknown command")
		}
	}
}

func readTime(date string) (*time.Time, *time.Time) {
	fmt.Println("=Range start (HH:mm:ss) :")
	range_start, err1 := reader.ReadString('\n')
	if cmdHandleErr(err1) {
		return nil, nil
	}
	range_start = strings.Replace(range_start, "\r\n", "", -1)
	range_start = date + "T" + range_start + "-00:00"
	timestart, terr := time.Parse(time.RFC3339, range_start)
	if cmdHandleErr(terr) {
		return nil, nil
	}

	fmt.Println("=Range end (HH:mm:ss) :")
	range_end, err2 := reader.ReadString('\n')
	if cmdHandleErr(err2) {
		return nil, nil
	}
	range_end = strings.Replace(range_end, "\r\n", "", -1)
	range_end = date + "T" + range_end + "-00:00"
	timeend, terr1 := time.Parse(time.RFC3339, range_end)
	if cmdHandleErr(terr1) {
		return nil, nil
	}
	return &timestart, &timeend
}

func readBroadcastType(start, end time.Time) *database.Plan {
	fmt.Println("=Broadcast type (playlist | silence | file) :")
	bcast_type, err := reader.ReadString('\n')
	if cmdHandleErr(err) {
		return nil
	}
	bcast_type = strings.Replace(bcast_type, "\r\n", "", -1)

	if bcast_type == "playlist" {
		fmt.Println("=Playlist ID")

		plarray, err0 := database.GetPlaylistsArrayObject()
		if cmdHandleErr(err0) {
			return nil
		}

		fmt.Print("(")
		var pllist string
		for _, plobj := range plarray {
			pllist += strconv.Itoa(plobj.Id) + "-\"" + plobj.Name + "\", "
		}
		pllist = strings.TrimSuffix(pllist, ", ")
		fmt.Print(pllist + ") :")
		fmt.Println()

		play_id, err1 := reader.ReadString('\n')
		if cmdHandleErr(err1) {
			return nil
		}
		play_id = strings.TrimSuffix(play_id, "\r\n")

		plan := database.Plan{
			Range: database.Range{
				Start: start,
				End:   end,
			},
			Type: database.BroadcastTypes{
				Playlist: database.PlaylistBroadcastType{
					PlaylistId: play_id,
					Active:     true,
				},
			},
		}
		return &plan
	} else if bcast_type == "silence" {
		plan := database.Plan{
			Range: database.Range{
				Start: start,
				End:   end,
			},
			Type: database.BroadcastTypes{
				Silence: database.SilenceBroadcastType{
					Active: true,
				},
			},
		}
		return &plan
	} else if bcast_type == "file" {
		var locations []string
		for {
			fmt.Println("=File location (<location> | end):")

			loc, err1 := reader.ReadString('\n')
			if cmdHandleErr(err1) {
				log.Println("Failed to read!")
				break
			}
			loc = strings.TrimSuffix(loc, "\r\n")
			if loc == "end" {
				break
			}
			locations = append(locations, loc)
		}

		plan := database.Plan{
			Range: database.Range{
				Start: start,
				End:   end,
			},
			Type: database.BroadcastTypes{
				File: database.FileBroadcastType{
					Location: locations,
					Active:   true,
				},
			},
		}
		return &plan
	}
	return nil
}

func cmdHandleErr(err error) bool {
	if err != nil {
		log.Println("Command could not be read!", err)
		return true
	}
	return false
}

func printPlan(plan database.Plan) {
	fmt.Println(plan.Range.Start.Format("15:04:05") + " - " + plan.Range.End.Format("15:04:05"))
	if plan.Type.Playlist.Active {
		playlist := database.GetPlaylistInfoObject(plan.Type.Playlist.PlaylistId)
		if playlist == nil {
			fmt.Println("  Playlist ID: " + plan.Type.Playlist.PlaylistId + " (<NONEXISTENT>)")
		} else {
			fmt.Println("  Playlist ID: " + plan.Type.Playlist.PlaylistId + " (" + playlist.Name + ")")
		}

	} else if plan.Type.File.Active {
		for _, loc := range plan.Type.File.Location {
			fmt.Println("  File location: " + loc)
		}
	} else if plan.Type.Silence.Active {
		fmt.Println("  Silence")
	}
	fmt.Println()
}

func printSong(song database.SongData) {
	songid := strconv.Itoa(song.SongId)
	votes := strconv.Itoa(song.VoteCount)
	fmt.Println("Song " + songid)
	fmt.Println("  Title:    " + song.Title)
	fmt.Println("  Authors:  " + song.Authors)
	fmt.Println("  YouTube:  " + song.YTId)
	fmt.Println("  Released: " + song.ReleaseDate.Format("2006-01-02"))
	fmt.Println("  Votes:    " + votes)
	fmt.Println("  added to library on " + song.DebutedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

}

func printPlaylist(playlist database.Playlist) {
	id := strconv.Itoa(playlist.Id)
	rank := strconv.Itoa(playlist.Rank)
	fmt.Println("Playlist " + id)
	fmt.Println("  Title:       " + playlist.Name)
	fmt.Println("  Description: " + playlist.Desc)
	fmt.Println("  Rank:        " + rank)
	fmt.Println("  added to library on " + playlist.DebutDate.Format("2006-01-02 15:04:05"))
	fmt.Println()

}

func printHelp(cmd string) {
	fmt.Println("Not enough args")
	if cmd == "schedule" {
		fmt.Println("schedule today")
		fmt.Println("schedule set <YYYY-MM-dd>")
		fmt.Println("schedule change <YYYY-MM-dd>")
	} else if cmd == "song" {
		fmt.Println("song list [page]")
		fmt.Println("song add")
		fmt.Println("song delete <id>")
	} else if cmd == "playlist" {
		fmt.Println("playlist list")
		fmt.Println("playlist create")
		fmt.Println("playlist get <id> [page]")
		fmt.Println("playlist delete <id>")
		fmt.Println("playlist addsong <playlistid> <songid>")
		fmt.Println("playlist remsong <playlistid> <songid>")
	}
	fmt.Println()
}
