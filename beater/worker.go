package beater

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var debugf = logp.MakeDebug("ecsbeat")

// Worker ...
type Worker struct {
	cmd         *Command
	ecsClusters *EcsClusters
	period      time.Duration
}

// NewWorker ...
func NewWorker(cmd *Command, ecsClusters *EcsClusters, period time.Duration) *Worker {
	return &Worker{cmd, ecsClusters, period}
}

// Start should be called only once in the life of a Worker.
func (w *Worker) Start(done <-chan struct{}) <-chan common.MapStr {
	debugf("Starting %s", w)
	defer debugf("Stopped %s", w)

	out := make(chan common.MapStr, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.startFetching(done, out)
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (w *Worker) startFetching(done <-chan struct{}, out chan<- common.MapStr) {
	debugf("Starting %s", w)
	defer debugf("Stopped %s", w)

	// Fetch immediately.
	err := w.fetch(done, out)
	if err != nil {
		logp.Err("%v", err)
	}

	// Start timer for future fetches.
	t := time.NewTicker(w.period)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			err := w.fetch(done, out)
			if err != nil {
				logp.Err("%v", err)
			}
		}
	}
}

// fetch does the actual work to query ECS
func (w *Worker) fetch(done <-chan struct{}, out chan<- common.MapStr) error {
	defer logp.Recover(fmt.Sprintf("recovered from panic while fetching "))

	// TODO, currently, each work for a particular type of query, shall we assign each work to one customer, or even every VDC?
	for _, ecs := range w.ecsClusters.EcsSlice {
		events, err := GenerateEvents(w.cmd, ecs.Config, ecs.Client)
		if err != nil {
			logp.Err("%v", err)
			continue
		}
		for _, event := range events {
			if !writeEvent(done, out, event) {
				return nil
			}
		}
	}
	return nil
}

func writeEvent(done <-chan struct{}, out chan<- common.MapStr, event common.MapStr) bool {
	select {
	case <-done:
		return false
	case out <- event:
		return true
	}
}
