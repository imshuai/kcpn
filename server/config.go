package main

import (
	"encoding/json"
	"os"
)

// Config for server
type Config struct {
	Listen string `json:"listen"`
	Crypt  string `json:"crypt"`
	Mode   string `json:"mode"`
	//AutoExpire   int    `json:"autoexpire"`
	//ScavengeTTL  int    `json:"scavengettl"`
	MTU         int `json:"mtu"`
	SndWnd      int `json:"sndwnd"`
	RcvWnd      int `json:"rcvwnd"`
	DataShard   int `json:"datashard"`
	ParityShard int `json:"parityshard"`
	DSCP        int `json:"dscp"`
	//NoComp       bool   `json:"nocomp"`
	AckNodelay   bool `json:"acknodelay"`
	NoDelay      int  `json:"nodelay"`
	Interval     int  `json:"interval"`
	Resend       int  `json:"resend"`
	NoCongestion int  `json:"nc"`
	SockBuf      int  `json:"sockbuf"`
	//KeepAlive    int    `json:"keepalive"`
}

func parseJSONConfig(config *Config, path string) error {
	file, err := os.Open(path) // For read access.
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}
