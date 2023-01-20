/*

MIT License

Copyright (c) 2018 Davis Davalos-DeLosh

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package fading

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/faiface/beep"
)

// Holds fadeItter and trackItter
/*
fadeItter - Is used to fade in and out
trackItter - Represents the position into a song
*/
var itters map[int][]float64

var CurFader *Fader

// fader is a type so that fader.Stream() can be used with proper parameters to run properly
type Fader struct {
	// Streamer to fade
	Streamer beep.Streamer
	// How long in samples to fade in, and to fade out
	TimeSpan float64
	// What the volume should be for the streamer
	Volume float64
	// How long the audio is, so that fading in and out works properly
	AudioLength float64
	// ID so that it can persist itterators between bits of slices
	Id int
	// edit by radio
	Stop bool
}

func init() {
	Release()
}

// For testing fading capabilities
func Release() { // - edit by radio: renamed func to Release for access
	// Necessary for itters map, otherwise there is a nil map error
	itters = make(map[int][]float64)
}

// Options for the CrossfadeSream function
type Options struct {
	TimeSpan time.Duration // How long to fade in, and to fade out
	Volume   float64       // What the volume should be for the streamer
}

type OwnStreamer struct {
	Faders []*Fader
	Pos    int
	Mixer  beep.Mixer
}

func (bs *OwnStreamer) Err() error {
	return nil
}

func (bs *OwnStreamer) Len() int {
	return len(bs.Faders)
}

func (bs *OwnStreamer) Position() int {
	return bs.Pos
}

func (bs *OwnStreamer) Seek(p int) error {
	if p < 0 || bs.Len() < p {
		return fmt.Errorf("buffer: seek position %v out of range [%v, %v]", p, 0, bs.Len())
	}
	bs.Pos = p
	return nil
}

func (bs *OwnStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	fader := bs.Faders[bs.Pos]
	ittrz := itters[bs.Pos]

	if len(ittrz) == 0 {
		log.Println("new stream")
		bs.Mixer.Add(beep.StreamerFunc(bs.Faders[bs.Pos].Stream))
	} else if len(ittrz) > 0 && ittrz[1] >= fader.AudioLength-fader.TimeSpan {
		bs.Mixer.Clear() // TODO make the Fader.Stream() phase out when skipping
		bs.Pos++
	}

	return bs.Mixer.Stream(samples)
}

// CrossfadeStream crossfades between all songs specified in files
// The sample-rates between the two streams must be the same, otherwise weird things might happen
// If opts is nil, then reasonable defaults are used
func CrossfadeStream(format beep.Format, opts *Options, streams ...beep.StreamSeeker) *OwnStreamer {
	timeSpan := time.Second * 9
	volume := 1.0
	if opts != nil {
		timeSpan = opts.TimeSpan
		volume = opts.Volume
	}

	// Streamer that will contain all files
	var streamer = &OwnStreamer{Faders: []*Fader{}, Mixer: beep.Mixer{}, Pos: 0}
	// Create 1000 samples of silence so that beep.Mix has a non-nil streamer to work with
	//var silence = beep.Silence(100)
	// The time span of the file previous to the one calculating on it. Used to get timing for crossfading right
	//var lastTimeSpan float64
	// Specifies how long the streamer is, so that timing for crossfading is correct
	var position float64
	// Iterate through all files specified to add them to streamer with proper crossfade
	for id, stream := range streams {
		// Create the set of parameters for it's stream function
		var faderStream = &Fader{Streamer: stream, Volume: volume, TimeSpan: float64(format.SampleRate.N(timeSpan)), AudioLength: float64(stream.Len()), Id: id, Stop: false}
		// Create streamer with fading applied
		//changedStreamer := beep.StreamerFunc(faderStream.Stream)
		// Create amount of silence before playing sound. Uses position, which by itself would make it play after the previous song. Subtracting lastTimeSpan makes a crossfade effect
		//silenceAmount := int(position - lastTimeSpan)

		// if id-1 < 0 && id != 0 {
		// 	silence = streamer.Streamers[id-1]
		// }
		// Keeps previous streamer, and adds the new streamer with the silence in the beginning so it doesn't play over other songs
		streamer.Faders = append(streamer.Faders, faderStream)
		//streamer.Streamers[id] = )
		//silence = beep.Silence(100)

		//streamer = beep.Mix(streamer, beep.Seq(beep.Silence(silenceAmount), changedStreamer))
		// Add position for next file
		position = position + faderStream.AudioLength
		// Set last time span to current time span for next file
		//lastTimeSpan = faderStream.TimeSpan
		// edit by radio
		//faderStream.Positions[id] = position
	}

	return streamer
}

var debug int64

// Stream edits streamer so that it fades
func (v *Fader) Stream(samples [][2]float64) (n int, ok bool) {
	// Determines if this specific streamer has been run before. If it hasn't then it needs to create fadeItter and trackItter for it
	if len(itters) < v.Id+1 {
		// Print ID of song
		//fmt.Println(v.id)
		CurFader = v // - edit by radio

		// Create fadeItter and trackItter for the ID, and assign them to defaults of 0
		itters[v.Id] = []float64{0, 0}
	}
	// Assign name to the map's ints for easier reading
	/*
		fadeItter - Is used to fade in and out
		trackItter - Represents the position into a song
	*/
	var fadeItter = &itters[v.Id][0]
	var trackItter = &itters[v.Id][1]

	// Use default streamer, and revise off of that
	n, ok = v.Streamer.Stream(samples)
	// Set gain to 0 if math.Pow fails
	gain := 0.0
	// Make gain work with the volume
	gain = math.Pow(1, v.Volume)
	// x1 is 0 and represents the start of the fade
	var x1 float64
	// The start of the fade should be silent, so y1 is 0
	var y1 float64
	// End point should be the TimeSpan set so that at the end of the TimeSpan, the gain is at requested value
	var x2 = v.TimeSpan
	// The requested gain, which will be played at the end of the TimeSpan
	var y2 = gain
	// Create the slope for a line representing this
	slopeUp := slopeCalc(x1, y1, x2, y2)
	//slopeDown := slopeCalc(x1, y2, x2, y1)
	// By default, sampleGain is the requested gain so between fadepoints, it is normal
	var sampleGain = gain
	// For each recieved sample, apply fade to it if necessary
	for i := range samples[:n] {
		// If the position in the track is after or at the time where it should begin to fade, then fade
		if *trackItter >= v.AudioLength-v.TimeSpan {
			// Slope-intercept form to get gain
			/*
				m					x 							+ 	b
				Calculated slope	The position in the fade		The y intercept of the gain, so that it fades down from the gain
			*/
			sampleGain = -slopeUp*float64(*fadeItter) + gain
			// Increment fade so that the next iteration will reduce the gain by more
			*fadeItter++
			// Prevents possible bug where the gain may become negative, which will result in the song's gain becoming high again
			if sampleGain < 0 {
				sampleGain = 0
			}
			// If the position of the track is before the specified TimeSpan, and the fadeItter isn't above the TimeSpan, begin to fade in.
		} else if *trackItter <= v.TimeSpan && slopeUp*float64(*fadeItter) <= gain {
			// Slope-intercept form to get gain
			/*
				m					x 							+ 	b
				Calculated slope	The position in the fade		0, because it is fading in from nothing
			*/
			sampleGain = slopeUp * float64(*fadeItter)
			// Increment fade so that the next iteration will reduce the gain by more
			*fadeItter++
		} else {
			// Ensures fadeItter isn't already high from fading in when it is time to fade out
			*fadeItter = 0
		}
		// Set the samples to the calculated gain
		samples[i][0] *= sampleGain
		samples[i][1] *= sampleGain
		// Increment trackItter to update position in track
		*trackItter++
	}

	if debug%int64(100) == 0 {
		log.Println("doing")
	}
	debug++

	// Return the samples with gain applied, and whether or not operations were successful
	return n, ok // edit by radio
}

// Calculates the slope between two points
func slopeCalc(x1 float64, y1 float64, x2 float64, y2 float64) float64 {
	return (y2 - y1) / (x2 - x1)
}
