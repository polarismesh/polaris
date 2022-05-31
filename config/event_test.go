package config

import (
	"log"
	"strconv"
	"sync"
	"testing"
)

func TestCenter_WatchEvent(t *testing.T) {
	c := NewEventCenter()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			cb := func(event Event) bool {
				event.Message = i
				log.Println("handler event: ", event.EventType, "msg:", event.Message)
				return true
			}

			eventType := "test_" + strconv.Itoa(i)
			log.Println("eventType2: ", eventType)
			c.WatchEvent(eventType, cb)
		}(i, &wg)

		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()

			eventType := "test_" + strconv.Itoa(i)
			log.Println("eventType: ", eventType)
			c.handleEvent(Event{
				EventType: eventType,
			})
		}(i, &wg)
	}

	wg.Wait()
	log.Println("test event watch end")
}
