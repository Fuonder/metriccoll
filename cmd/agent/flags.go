package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
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
	version  = "0.1.15"
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

type NetAddress struct {
	IPAddr string
	Port   int
}

func (n *NetAddress) String() string {
	return fmt.Sprintf("%s:%d", n.IPAddr, n.Port)
}

func (n *NetAddress) Set(value string) error {
	values := strings.Split(value, ":")
	if len(values) != 2 {
		return fmt.Errorf("%w: \"%s\"", ErrNotFullIP, value)
	}
	n.IPAddr = values[0]
	if n.IPAddr == "" {
		return fmt.Errorf("%w: \"%s\"", ErrInvalidIP, values[0])
	}
	var err error
	n.Port, err = strconv.Atoi(values[1])
	if err != nil {
		return fmt.Errorf("%w: \"%s\"", ErrInvalidPort, values[1])
	}
	return nil
}

type CliOptions struct {
	NetAddr        NetAddress
	ReportInterval time.Duration
	PollInterval   time.Duration
	HashKey        string
	RateLimit      time.Duration
}

func (o *CliOptions) String() string {
	return fmt.Sprintf("netAddr:%s, reportInterval:%s, pollInterval:%s, hashKey:%s, rateLimit: %d",
		o.NetAddr.String(),
		o.ReportInterval,
		o.PollInterval,
		o.HashKey,
		o.RateLimit)
}

var (
	CliOpt = CliOptions{
		NetAddr: NetAddress{
			IPAddr: "localhost",
			Port:   8080},
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
		HashKey:        "",
		RateLimit:      1 * time.Second,
	}
	netAddr = &NetAddress{
		IPAddr: "localhost",
		Port:   8080,
	}
	pInterval int64 = 2
	rInterval int64 = 10
	rate      int64 = 1
)

func validateIntervalString(interval string) error {
	i, err := strconv.Atoi(interval)
	if err != nil {
		return fmt.Errorf("malformed interval value: \"%s\": %w", interval, err)
	}
	if i <= 0 {
		return fmt.Errorf("interval out of range: %s", interval)
	}
	return nil
}

func validateIntervalInt64(interval int64) error {
	if interval <= 0 {
		return fmt.Errorf("interval out of range: %d", interval)
	}
	return nil
}

func parseFlags() error {
	flag.Usage = usage
	flag.Var(netAddr, "a", "ip and port of server in format <ip>:<port>")
	flag.Int64Var(&pInterval, "p", 2, "interval of collecting metrics in secs")
	flag.Int64Var(&rInterval, "r", 10, "interval of reports in secs")
	flag.StringVar(&CliOpt.HashKey, "k", "", "key for hash")
	flag.Int64Var(&rate, "l", 1, "rate limit")

	flag.Parse()
	var err error

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		err = CliOpt.NetAddr.Set(envRunAddr)
		if err != nil {
			return err
		}
	} else if netAddr != nil {
		CliOpt.NetAddr = *netAddr
	}

	if envRInterval := os.Getenv("REPORT_INTERVAL"); envRInterval != "" {
		err = validateIntervalString(envRInterval)
		if err != nil {
			return fmt.Errorf("REPORT_INTERVAL: %w", err)
		}
		CliOpt.ReportInterval, err = time.ParseDuration(envRInterval + "s")
		if err != nil {
			return fmt.Errorf("REPORT_INTERVAL: %w", err)
		}
	} else {
		err = validateIntervalInt64(rInterval)
		if err != nil {
			return fmt.Errorf("flag -r: %w", err)
		}
		CliOpt.ReportInterval = time.Duration(rInterval) * time.Second
	}

	if envPInterval := os.Getenv("POLL_INTERVAL"); envPInterval != "" {
		err = validateIntervalString(envPInterval)
		if err != nil {
			return fmt.Errorf("POLL_INTERVAL: %w", err)
		}
		CliOpt.PollInterval, err = time.ParseDuration(envPInterval + "s")
		if err != nil {
			return fmt.Errorf("POLL_INTERVAL: %w", err)
		}
	} else {
		err = validateIntervalInt64(pInterval)
		if err != nil {
			return fmt.Errorf("flag -p: %w", err)
		}
		CliOpt.PollInterval = time.Duration(pInterval) * time.Second
	}

	if envHashKey := os.Getenv("KEY"); envHashKey != "" {
		CliOpt.HashKey = envHashKey
	}

	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		err = validateIntervalString(envRateLimit)
		if err != nil {
			return fmt.Errorf("RATE_LIMIT: %w", err)
		}

		CliOpt.RateLimit, err = time.ParseDuration(envRateLimit + "s")
		if err != nil {
			return fmt.Errorf("RATE_LIMIT: %w", err)
		}
	} else {
		err = validateIntervalInt64(rate)
		if err != nil {
			return fmt.Errorf("flag -l: %w", err)
		}
		CliOpt.RateLimit = time.Duration(rate) * time.Second
	}

	return nil
}
