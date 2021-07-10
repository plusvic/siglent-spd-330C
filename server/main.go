package main

import (
	"container/list"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Measurement struct {
	Timestamp time.Time
	Value     float32
}

type MeasurementsList struct {
	sync.Mutex
	*list.List
}

var (
	measurements MeasurementsList
	upgrader     websocket.Upgrader
)

func serveWebsocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var ms [10000]Measurement

	for range ticker.C {
		m := measurements.Back()
		for i := len(ms) - 1; i >= 0 && m != nil; i-- {
			ms[i] = m.Value.(Measurement)
			m = m.Prev()
		}
		err := c.WriteJSON(ms)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func trackMeasurements() {
	samplingPeriod := 10 * time.Millisecond
	numSamples := 10000
	now := time.Now()

	// Initialize samples with 0 values.
	measurements.Lock()
	for i := 0; i < numSamples; i++ {
		fmt.Println(now.Add(-time.Duration(i) * time.Millisecond))
		measurements.PushFront(Measurement{
			Timestamp: now.Add(-time.Duration(i) * samplingPeriod),
			Value:     0,
		})
	}
	measurements.Unlock()

	ticker := time.NewTicker(samplingPeriod)
	for t := range ticker.C {
		measurements.Lock()
		measurements.Remove(measurements.Front())
		measurements.PushBack(Measurement{
			Timestamp: t,
			Value:     rand.Float32(),
		})
		measurements.Unlock()
	}
}

func main() {
	measurements = MeasurementsList{List: list.New()}
	go trackMeasurements()
	fs := http.FileServer(http.Dir("../ui"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", serveWebsocket)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
