package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/BenLubar/espeak"
	"golang.org/x/mobile/exp/audio/al"
)

var (
	audioData   []int16
	mouthEvents []mouthEvent

	source al.Source
)

type voiceRequest struct {
	Text string `json:"text"`
}

var voiceRequests chan voiceRequest

func processVoiceRequests() {
	for req := range voiceRequests {
		log.Println("servicing request", req)
		talk(req.Text)
		log.Println("done")
	}
}

// talk plays a voice synthesis of the given string of text. This function
// blocks until finished. It is not thread-safe.
func talk(text string) error {
	var uid uint
	_ = uid

	// Tell espeak to start processing text
	err := espeak.Synth(text, 0, espeak.PosCharacter, uint(len(text)-1), 0, &uid, nil)
	if err != nil {
		return err
	}

	// Wait until processing is done
	err = espeak.Synchronize()
	if err != nil {
		return err
	}

	// Play the results
	play()

	// Clear all old data
	audioData = nil
	mouthEvents = nil

	return nil
}

// play plays the audio loaded in audioData and executes movements stored in
// mouthEvents.
func play() {
	buffer := al.GenBuffers(1)[0]
	defer al.DeleteBuffers(buffer)

	var rawData bytes.Buffer
	err := binary.Write(&rawData, binary.LittleEndian, audioData)
	if err != nil {
		panic(err)
	}

	buffer.BufferData(al.FormatMono16, rawData.Bytes(), 44100/2)

	source.QueueBuffers(buffer)
	al.PlaySources(source)

	for source.State() == al.Playing {
	}

	source.UnqueueBuffers(buffer)
}

// schedMouthMovements schedules mouth movements to be done asynchronously
// at the right time.
func schedMouthMovements() {
	for _, event := range mouthEvents {
		e := event
		time.AfterFunc(time.Duration(event.pos)*time.Millisecond, func() {
			switch e.typ {
			case mouthOpen:
				fmt.Println("open")
			case mouthClose:
				fmt.Println("close")
			}
		})
	}
}

// synthCallback is called by eSpeak while TTS processing. In here is where
// audio data is collected and mouth events are recorded.
func synthCallback(wav []int16, events []espeak.Event) bool {
	if len(wav) == 0 {
		return true
	}

	lastMouthState := mouthClose
	for _, event := range events {
		switch t := event.(type) {
		case espeak.WordEvent:
			if lastMouthState == mouthOpen {
				// Close the mouth before opening it again
				mouthEvents = append(mouthEvents, mouthEvent{
					pos: t.AudioPosition - 20,
					typ: mouthClose})
			}

			// Open the mouth to speak
			mouthEvents = append(mouthEvents, mouthEvent{
				pos: t.AudioPosition,
				typ: mouthOpen})
			lastMouthState = mouthOpen

		case espeak.EndEvent:
			// Close the mouth after we're done speaking
			mouthEvents = append(mouthEvents, mouthEvent{
				pos: t.AudioPosition,
				typ: mouthClose})
			lastMouthState = mouthClose
		}
	}

	// Save audio data to be played later
	audioData = append(audioData, wav...)

	return false
}

// initVoiceSynth initializes all voice synthesis related state.
func initVoiceSynth() {
	voiceRequests = make(chan voiceRequest)

	// Open audio device
	err := al.OpenDevice()
	if err != nil {
		panic(err)
	}

	// Open up a source to play to
	source = al.GenSources(1)[0]

	// Intialize voice synthesis
	_, err = espeak.Initialize(espeak.AudioOutputSynchronous, bufferLength, "", 0)
	if err != nil {
		panic(err)
	}
	espeak.SetSynthCallback(synthCallback)

	// Configure a voice
	espeak.SetVoiceByProperties("klatt", "en", espeak.Male, 80, 1)
	err = espeak.SetRate(120)
	if err != nil {
		panic(err)
	}

	log.Println("initialized OpenAL and eSpeak")
}

// cleanUpVoiceSynth cleans up any state from voice synthesis initialization.
func cleanUpVoiceSynth() {
	al.DeleteSources(1, source)
	al.CloseDevice()
}
