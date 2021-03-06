// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

// Customer ...
type Customer struct {
	CustomerName       string        `config:"customername"`
	Username           string        `config:"username"`
	Password           string        `config:"password"`
	TokenExpiry        time.Duration `config:"tokenexpiry"`
	ReqTimeOut         time.Duration `config:"reqtimeout"`
	BlockDuration      time.Duration `config:"blockduration"`
	CfgRefreshInterval time.Duration `config:"cfgrefreshinterval"`
	VDCs               []*struct {
		VdcName string `config:vdcname`
		Nodes   []*struct {
			IP string `config:"host"`
		} `config:"nodes"`
	} `config:"vdcs"`
}

// Config ...
type Config struct {
	Period   time.Duration `config:"period"`
	Once     bool          `config:"once"`
	Commands []*struct {
		URI      string        `config:"uri"`
		Type     string        `config:"type"`
		Level    string        `config:"level"`
		Interval time.Duration `config:"interval"`
		Enabled  bool          `config:"enabled"`
	} `config:"commands"`
	Customers []*Customer `config:"customers"`
}

var DefaultConfig = Config{
	Period: 60 * time.Second,
}
