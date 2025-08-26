package main

import (
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100
const seconds = 1

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate()
	buffer := make([]float32, sampleRate*seconds)
	stream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(buffer), func(in []float32) {
		for i := range buffer {
			buffer[i] = in[i]
			fmt.Printf("%f\n", buffer[i])
		}
	})
	if err != nil {
		panic(err)
	}
	stream.Start()
	defer stream.Close()
	http.HandleFunc("/audio", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}

		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "audio/wave")
		for {
			binary.Write(w, binary.BigEndian, &buffer)
			flusher.Flush() // Trigger "chunked" encoding and send a chunk...
			return
		}
	})

	http.ListenAndServe(":8080", nil)
}
