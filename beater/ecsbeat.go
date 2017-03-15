package beater

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/yangb8/ecsbeat/config"
)

// Ecsbeat ...
type Ecsbeat struct {
	done        chan struct{}
	config      config.Config
	client      publisher.Client
	ecsClusters *EcsClusters
	workers     []*Worker
}

// New is to create beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	ec := NewEcsClusters(config)

	bt := &Ecsbeat{
		done:        make(chan struct{}),
		config:      config,
		ecsClusters: ec,
	}

	// Init Cofingurations for Ecs Clusters
	bt.ecsClusters.Refresh()

	for _, c := range ec.Cmds {
		w := NewWorker(c, bt.ecsClusters)
		bt.workers = append(bt.workers, w)
	}

	return bt, nil
}

// Run ...
func (bt *Ecsbeat) Run(b *beat.Beat) error {
	logp.Info("ecsbeat is running! Hit CTRL-C to stop it.")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartRefreshConfig(bt.ecsClusters, bt.done)
	}()

	bt.client = b.Publisher.Connect()

	var cs []<-chan common.MapStr
	for _, w := range bt.workers {
		cs = append(cs, w.Start(bt.done))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		PublishChannels(bt.client, cs...)
	}()
	// Wait for StartRefreshConfig exits and PublishChannels to stop publishing
	wg.Wait()
	return nil
}

// Stop ...
func (bt *Ecsbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
