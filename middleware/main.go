package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type ConnectionPool struct {
	bufferChannelMap map[chan []byte]struct{}
	mu               sync.Mutex
}

func (cp *ConnectionPool) AddConnection(bufferChannel chan []byte) {
	defer cp.mu.Unlock()
	cp.mu.Lock()
	cp.bufferChannelMap[bufferChannel] = struct{}{}
}

func (cp *ConnectionPool) DeleteConnection(bufferChannel chan []byte) {
	defer cp.mu.Unlock()
	cp.mu.Lock()
	delete(cp.bufferChannelMap, bufferChannel)
}

func (cp *ConnectionPool) Broadcast(buffer []byte) {
	defer cp.mu.Unlock()
	cp.mu.Lock()
	for bufferChannel, _ := range cp.bufferChannelMap {
		clonedBuffer := make([]byte, 4096)
		copy(clonedBuffer, buffer)
		select {
		case bufferChannel <- clonedBuffer:
		default:
		}
	}
}

func NewConnectionPool() *ConnectionPool {
	bufferChannelMap := make(map[chan []byte]struct{})
	return &ConnectionPool{bufferChannelMap: bufferChannelMap}
}

func stream(connectionPool *ConnectionPool, content []byte) {
	buffer := make([]byte, len(content))
	for {
		tempfile := bytes.NewReader(content)
		// clear() is a builtin function introduced in go 1.21.
		// Reinitialize the buffer if on a lower version.
		buffer = make([]byte, len(content))
		ticker := time.NewTicker(time.Millisecond * 250)
		for range ticker.C {
			_, err := tempfile.Read(buffer)
			if err == io.EOF {
				ticker.Stop()
				break
			}
			connectionPool.Broadcast(buffer)
		}
	}
}

func getAudioBuffer(buffer []byte) {
	for {
		resp, err := http.Get("http://localhost:8080/audio")
		chk(err)
		body, _ := io.ReadAll(resp.Body)
		responseReader := bytes.NewReader(body)
		binary.Read(responseReader, binary.BigEndian, &buffer)
	}
}

const sampleRate = 44100
const seconds = 1

func main() {
	buffer := make([]byte, sampleRate*seconds)
	connPool := NewConnectionPool()

	go stream(connPool, buffer)
	go getAudioBuffer(buffer)

	http.HandleFunc("/audio", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "audio/wave")

		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Println("Could not create flusher")
		}
		bufferChannel := make(chan []byte)
		connPool.AddConnection(bufferChannel)
		log.Printf("%s has connected\n", r.Host)
		for {
			buf := <-bufferChannel
			if _, err := w.Write(buf); err != nil {
				connPool.DeleteConnection(bufferChannel)
				log.Printf("%s's connection has been closed\n", r.Host)
				return
			}
			flusher.Flush()
		}
	})
	log.Println("Listening on port 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))

}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
