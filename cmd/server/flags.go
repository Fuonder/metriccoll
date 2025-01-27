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
	version  = "0.1.9"
	progName = "Fuonder's ya-practicum server"
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

var (
	ErrNotFullIP   = errors.New("given ip address and port incorrect")
	ErrInvalidIP   = errors.New("incorrect ip address")
	ErrInvalidPort = errors.New("incorrect port number")
)

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

var (
	netAddr = &netAddress{
		ipaddr: "localhost",
		port:   8080,
	}
	flagLogLevel        string
	flagStoreInterval   time.Duration
	sIntervalInt64      int64 = 300
	flagFileStoragePath string
	flagRestore         bool
)

func validateIntervalString(interval string) error {
	i, err := strconv.Atoi(interval)
	if err != nil {
		return fmt.Errorf("malformed interval value: \"%s\": %w", interval, err)
	}
	if i < 0 {
		return fmt.Errorf("interval out of range: %s", interval)
	}
	return nil
}

func validateIntervalInt64(interval int64) error {
	if interval < 0 {
		return fmt.Errorf("interval out of range: %d", interval)
	}
	return nil
}

func parseFlags() error {
	flag.Usage = usage
	flag.Var(netAddr, "a", "ip and port of server in format <ip>:<port>")
	flag.StringVar(&flagLogLevel, "l", "info", "loglevel")
	flag.Int64Var(&sIntervalInt64, "i", 300, "interval for metrics dump in seconds")
	flag.StringVar(&flagFileStoragePath, "f", "./metrics.dump", "Path to metrics dump file")
	flag.BoolVar(&flagRestore, "r", true, "load metrics from dump on start")

	flag.Parse()
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		err := netAddr.Set(envRunAddr)
		if err != nil {
			return err
		}
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		err := validateIntervalString(envStoreInterval)
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
		flagStoreInterval, err = time.ParseDuration(envStoreInterval + "s")
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
	} else {
		err := validateIntervalInt64(sIntervalInt64)
		if err != nil {
			return fmt.Errorf("flag -i: %w", err)
		}
		flagStoreInterval = time.Duration(sIntervalInt64) * time.Second
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		flagFileStoragePath = envFileStoragePath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		var err error
		flagRestore, err = strconv.ParseBool(envRestore)
		if err != nil {
			return fmt.Errorf("invalid RESTORE value: %w", err)
		}
	}
	return nil
}
