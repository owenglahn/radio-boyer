package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100
const seconds = 1

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate()
	buffer := make([]float32, sampleRate*seconds)

	stream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(buffer), func(out []float32) {
		resp, err := http.Get("http://localhost:8080/audio")
		chk(err)
		body, _ := io.ReadAll(resp.Body)
		responseReader := bytes.NewReader(body)
		binary.Read(responseReader, binary.BigEndian, &buffer)
		for i := range out {
			fmt.Printf("%f\n", buffer[i])
			out[i] = buffer[i]
		}
	})
	chk(err)
	chk(stream.Start())
	time.Sleep(time.Second * 40)
	chk(stream.Stop())
	defer stream.Close()

	if err != nil {
		fmt.Println(err)
	}

}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
