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
}

// NewWorker ...
func NewWorker(cmd *Command, ecsClusters *EcsClusters) *Worker {
	return &Worker{cmd, ecsClusters}
}

// Start should be called only once in the life of a Worker.
func (w *Worker) Start(done <-chan struct{}, once bool) <-chan common.MapStr {
	debugf("Starting %s", w)
	defer debugf("Stopped %s", w)

	out := make(chan common.MapStr, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.startFetching(done, out, once)
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (w *Worker) startFetching(done <-chan struct{}, out chan<- common.MapStr, once bool) {
	debugf("Starting %s", w)
	defer debugf("Stopped %s", w)

	// Fetch immediately.
	err := w.fetch(done, out)
	if err != nil {
		logp.Err("%v", err)
	}

	if once {
		return
	}

	// Start timer for future fetches.
	t := time.NewTicker(w.cmd.Interval)
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
		torun, err := GenerateEvents(w.cmd, ecs.Config, ecs.Client, done, out)
		if !torun {
			return nil
		}
		if err != nil {
			logp.Err("%v", err)
			continue
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
