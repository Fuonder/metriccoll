package main

import "fmt"

type config struct {
	ipaddr string
	port   string
}

func newConfig(ip string, port string) (*config, error) {
	cfg := config{
		ipaddr: ip,
		port:   port,
	}
	return &cfg, nil
}

func (c config) fullAddr() string {
	return fmt.Sprintf("%s:%s", c.ipaddr, c.port)
}
