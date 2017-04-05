package beater

import (
	"sync"
	"time"

	"github.com/yangb8/ecsbeat/config"
	"github.com/yangb8/ecsbeat/ecs"
)

// GetEcsFromConfig ...
func GetEcsFromConfig(c *config.Customer) *ecs.Ecs {
	Vdcs := make(map[string]*ecs.Vdc)
	for _, vdc := range c.VDCs {
		nodes := make([]string, len(vdc.Nodes))
		for i, n := range vdc.Nodes {
			nodes[i] = n.IP
		}
		Vdcs[vdc.VdcName] = ecs.NewVdc(vdc.VdcName, nodes)
	}
	return ecs.NewEcs(Vdcs)
}

// GetClusterConfig ...
func GetClusterConfig(c *config.Customer) *ClusterConfig {
	Vdcs := make(map[string]*Vdc)
	for _, vdc := range c.VDCs {
		Vdcs[vdc.VdcName] = &Vdc{
			ConfigName: vdc.VdcName,
			// content of NodeInfo will be filled by Refresh()
			NodeInfo: make(map[string]*Node),
		}
	}
	return &ClusterConfig{c.CustomerName, c.CfgRefreshInterval, Vdcs}
}

// ClusterConfig ...
type ClusterConfig struct {
	CustomerName string
	CfgRefresh   time.Duration
	Vdcs         map[string]*Vdc
}

// Vdc ...
type Vdc struct {
	ConfigName       string           `json:"ecs-config-name"`
	ID               string           `json:"ecs-vdc-id"`
	Name             string           `json:"ecs-vdc-name"`
	StoragepoolID    string           `json:"ecs-storagepool-id"`
	StoragepoolName  string           `json:"ecs-storagepool-name"`
	LastUpdatedDate  string           `json:"lastUpdatedDate"`
	ManualUpdateOnly bool             `json:"manualUpdateOnly"`
	NodeInfo         map[string]*Node `json:"nodeInfo"`
	mutex            sync.RWMutex
}

// Update ...
func (v *Vdc) Update(cname, id, name string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.ConfigName = cname
	v.ID = id
	v.Name = name
}

// Get ...
func (v *Vdc) Get() (string, string, string) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.ConfigName, v.ID, v.Name
}

// Node ...
type Node struct {
	IP      string `json:"ecs-node-ip"`
	Name    string `json:"ecs-node-Name"`
	Version string `json:"ecs-version"`
	mutex   sync.RWMutex
}

// Update ...
func (n *Node) Update(ip, name, version string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.IP = ip
	n.Name = name
	n.Version = version
}

// Get ...
func (n *Node) Get() (string, string, string) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.IP, n.Name, n.Version
}

func convertMapStringInterface(val interface{}) map[string]interface{} {
	result, ok := val.(map[string]interface{})
	if ok {
		return result
	}

	m := val.(map[interface{}]interface{})
	result = make(map[string]interface{})

	for k, v := range m {
		result[k.(string)] = v
	}

	return result
}

// Command ...
type Command struct {
	URI      string
	Type     string
	Level    string
	Interval time.Duration
}
