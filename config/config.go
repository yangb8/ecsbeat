// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period   time.Duration `config:"period"`
	Commands []*struct {
		URI      string        `config:"uri"`
		Type     string        `config:"type"`
		Level    string        `config:"level"`
		Interval time.Duration `config:"interval"`
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
				IP string `config:"host"`
			} `config:"nodes"`
		} `config:"vdcs"`
	} `config:"customers"`
}

var DefaultConfig = Config{
	Period: 60 * time.Second,
}
