package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotFullIP       = errors.New("given ip address and port incorrect")
	ErrInvalidIP       = errors.New("incorrect ip address")
	ErrInvalidPort     = errors.New("incorrect port number")
	ErrInvalidArgument = errors.New("invalid argument")
)

var (
	version  = "0.1.4"
	progName = "Fuonder's ya-practicum client"
	source   = "https://github.com/Fuonder/metriccoll"
)

var usage = func() {
	fmt.Fprintf(flag.CommandLine.Output(), "%s\nSource code:\t%s\nVersion:\t%s\nUsage of %s:\n",
		progName,
		source,
		version,
		progName)
	flag.PrintDefaults()
}

type netAddress struct {
	ipaddr string
	port   int
}

func (n *netAddress) String() string {
	return fmt.Sprintf("%s:%d", n.ipaddr, n.port)
}

func (n *netAddress) Set(value string) error {
	values := strings.Split(value, ":")
	if len(values) != 2 {
		return fmt.Errorf("%w: \"%s\"", ErrNotFullIP, value)
	}
	n.ipaddr = values[0]
	if n.ipaddr == "" {
		return fmt.Errorf("%w: \"%s\"", ErrInvalidIP, values[0])
	}
	var err error
	n.port, err = strconv.Atoi(values[1])
	if err != nil {
		return fmt.Errorf("%w: \"%s\"", ErrInvalidPort, values[1])
	}
	return nil
}

type options struct {
	netAddr        netAddress
	reportInterval time.Duration
	pollInterval   time.Duration
}

func (o *options) String() string {
	return fmt.Sprintf("netAddr:%s, reportInterval:%s, pollInterval:%s",
		o.netAddr.String(),
		o.reportInterval,
		o.pollInterval)
}

var (
	opt = options{
		netAddr: netAddress{
			ipaddr: "localhost",
			port:   8080},
		reportInterval: 10 * time.Second,
		pollInterval:   2 * time.Second,
	}
	netAddr = &netAddress{
		ipaddr: "localhost",
		port:   8080,
	}
	pInterval int64 = 2
	rInterval int64 = 10
)

func parseFlags() error {
	flag.Usage = usage
	flag.Var(netAddr, "a", "ip and port of server in format <ip>:<port>")
	flag.Int64Var(&pInterval, "p", 2, "interval of collecting metrics in secs")
	flag.Int64Var(&rInterval, "r", 10, "interval of reports in secs")

	flag.Parse()
	opt.netAddr = *netAddr

	if pInterval <= 0 {
		return fmt.Errorf("%w: \"%d\"", ErrInvalidArgument, pInterval)
	}
	if rInterval <= 0 {
		return fmt.Errorf("%w: \"%d\"", ErrInvalidArgument, rInterval)
	}

	opt.pollInterval = time.Duration(pInterval) * time.Second
	opt.reportInterval = time.Duration(rInterval) * time.Second
	return nil
}
