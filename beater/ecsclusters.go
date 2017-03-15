package beater

import (
	"sync"
	"time"

	"github.com/yangb8/ecsbeat/config"
	"github.com/yangb8/ecsbeat/ecs"
)

// EcsCluster ...
type EcsCluster struct {
	CustomerName string
	CfgRefresh   time.Duration
	Config       *ClusterConfig
	Client       *ecs.MgmtClient
}

// Refresh ...
func (e *EcsCluster) Refresh() {
	for vname, ventry := range e.Config.Vdcs {
		if vdcResp, err := ecs.GetLocalVDC(e.Client, vname); err == nil {
			ventry.Update(vname, vdcResp.ID, vdcResp.Name)
		}
		if nodesResp, err := ecs.GetNodes(e.Client, vname); err == nil {
			for _, n := range nodesResp.Node {
				if _, ok := ventry.NodeInfo[n.IP]; ok {
					ventry.NodeInfo[n.IP].Update(n.IP, n.Nodename, n.Version)
				}
			}
		}
		// TODO log error
	}
}

// NewEcsCluster ...
func NewEcsCluster(customer string, config config.Config) *EcsCluster {
	properties := config.Customers[customer]
	return &EcsCluster{
		Config: GetClusterConfig(customer, config),
		Client: ecs.NewMgmtClient(
			"ecs",
			properties.Username,
			properties.Password,
			GetEcsFromConfig(customer, config),
			properties.ReqTimeOut,
			properties.BlockDuration),
	}
}

// EcsClusters ...
type EcsClusters struct {
	Cmds     []*Command
	EcsSlice []*EcsCluster
}

// NewEcsClusters ...
func NewEcsClusters(config config.Config) *EcsClusters {
	ec := EcsClusters{}
	for _, c := range config.Commands {
		interval := config.Period
		// Overwrite default interval by command level interval
		if c.Interval > 0 {
			interval = c.Interval
		}
		ec.Cmds = append(ec.Cmds, &Command{c.URI, c.Type, c.Level, interval})
	}

	for customer := range config.Customers {
		ec.EcsSlice = append(ec.EcsSlice, NewEcsCluster(customer, config))
	}

	return &ec
}

// Refresh ...
func (ec *EcsClusters) Refresh() {
	for _, cluster := range ec.EcsSlice {
		cluster.Refresh()
	}
}

// StartRefreshConfig ...
func StartRefreshConfig(ec *EcsClusters, done <-chan struct{}) {
	var wg sync.WaitGroup
	for _, ecs := range ec.EcsSlice {
		if ecs.CfgRefresh > 0 {
			wg.Add(1)
			go func(e *EcsCluster, done <-chan struct{}) {
				defer wg.Done()
				t := time.NewTicker(e.CfgRefresh)
				defer t.Stop()
				for {
					select {
					case <-done:
						return
					case <-t.C:
						e.Refresh()
					}
				}

			}(ecs, done)
		}
	}
	wg.Wait()
}
