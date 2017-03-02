// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period   time.Duration `config:"period"`
	Commands []*struct {
		URI   string `config:"uri"`
		Type  string `config:"type"`
		Level string `config:"level"`
	} `config:"commands"`
	Customers map[string]struct {
		Username           string        `config:"username"`
		Password           string        `config:"password"`
		TokenExpiry        time.Duration `config:"tokenexpiry"`
		ReqTimeOut         time.Duration `config:"reqtimeout"`
		BlockDuration      time.Duration `config:"blockduration"`
		CfgRefreshInterval time.Duration `config:"cfgrefreshinterval"`
		VDCs               map[string]*struct {
			Nodes []*struct {
				IP string `config:"ip"`
			} `config:"nodes"`
		} `config:"vdcs"`
	} `config:"customers"`
}

//      blockduration: 600s         # how long a node shall be put into blacklist if request to this node times out. 0s for not blocking
//      vdcs:
//         - name: VDC1            # VDC name, could be anything as long as each VDC has different name
//           nodes:
//             - 10.1.83.51        # mgmt IP, add IPs of all nodes in this VDC below
//             - 10.1.83.52        # mgmt IP
//         - name: VDC2            # VDC name
//           nodes:
//             - 10.1.83.53        # mgmt IP, add IPs of all nodes in this VDC below
//             - 10.1.83.54        # mgmt IP
//      cfgrefreshinterval: 3600s  # How frequent to update VDC and node names. Generally, default value is good enough because these info is almost never changed
//    DELL:
//      username: root
//      password: ChangeMe
//      tokenexpiry: 3500s
//      reqtimeout: 30s
//      blockduration: 600s
//      vdcs:
//         - name: EAST
//           nodes:
//             - 10.1.83.51
//             - 10.1.83.52
//         - name: WEST
//           nodes:
//             - 10.1.83.53
//             - 10.1.83.54
//      cfgrefreshinterval: 3600s}

var DefaultConfig = Config{
	Period: 60 * time.Second,
}
