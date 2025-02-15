package main

import (
	"context"
	"ip-blackcage/event/ipevent"
	"log"
)

func main() {
	ev, err := ipevent.NewIPEventReader(ipevent.WithEnablePortVisit([]uint16{2048}), ipevent.WithListenInterface("br0"))
	if err != nil {
		panic(err)
	}
	ch, err := ev.Open(context.Background())
	if err != nil {
		panic(err)
	}
	for c := range ch {
		log.Printf("t:%d ip:%s", c.Timestamp(), c.Data().(string))
	}
}
