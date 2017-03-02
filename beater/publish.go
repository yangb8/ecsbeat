package beater

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

// PublishChannels use client to read events from all incoming channels
func PublishChannels(client publisher.Client, cs ...<-chan common.MapStr) {
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(ch <-chan common.MapStr) {
			defer wg.Done()
			for event := range ch {
				client.PublishEvent(event)
			}
		}(c)
	}
	wg.Wait()
}
