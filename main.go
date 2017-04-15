package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"

	"github.com/pkg/errors"
)

const bufferLength = 4096

type mouthEventType int

const (
	mouthOpen mouthEventType = iota
	mouthClose
)

type mouthEvent struct {
	pos int
	typ mouthEventType
}

func main() {
	initVoiceSynth()
	defer cleanUpVoiceSynth()

	http.HandleFunc("/", onVoiceRequest)

	// Start the server in the background
	go func() {
		log.Fatalln(http.ListenAndServe(":80", nil))
	}()

	// Start the voice request processing service
	processVoiceRequests()
}

// onVoiceRequest handles an HTTP request to have something spoken.
func onVoiceRequest(w http.ResponseWriter, r *http.Request) {
	reqData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to read request body")
		http.Error(w, err.Error(), 500)
		log.Println(err)
		return
	}

	// Parse the request
	var voiceReq voiceRequest
	err = json.Unmarshal(reqData, &voiceReq)
	if err != nil {
		err = errors.Wrap(err, "failed to parse request")
		http.Error(w, err.Error(), 500)
		log.Println(err)
		return
	}

	// Queue up the voice request
	voiceRequests <- voiceReq
}

func init() {
	// Always safer to do this when using C bindings
	runtime.LockOSThread()
}
