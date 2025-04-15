package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/storage"
)

var (
	version  = "0.1.15"
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

type Flags struct {
	NetAddress      netAddress
	LogLevel        string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	HashKey         string
}

func (f *Flags) String() string {
	return fmt.Sprintf("netAddr: %s, "+
		"LogLevel: %s, "+
		"StoreInterval: %s, "+
		"FileStoragePath: %s, "+
		"Restore: %v, "+
		"DatabaseDSN: %s, "+
		"HashKey: %s",
		f.NetAddress.String(),
		f.LogLevel,
		f.StoreInterval.String(),
		f.FileStoragePath,
		f.Restore,
		f.DatabaseDSN,
		f.HashKey,
	)
}

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

func checkPathWritable(path string) error {
	if path == "" {
		return fmt.Errorf("path can not be empty")
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("can not create file \"%s\": %w", path, err)
			}
			defer file.Close()
		} else {
			return fmt.Errorf("can not get information about path \"%s\": %w", path, err)
		}
	}

	file, err := os.OpenFile(path, os.O_RDWR, storage.OsAllRw)
	if err != nil {
		return fmt.Errorf("can not open file in Write mode: %w", err)
	}
	defer file.Close()

	return nil
}

var (
	FlagsOptions = Flags{
		NetAddress: netAddress{
			ipaddr: "localhost",
			port:   8080},
		LogLevel:        "info",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "./metrics.dump",
		Restore:         true,
		DatabaseDSN:     "postgres://videos:12345678@localhost:5432/videos?sslmode=disable",
		HashKey:         "",
	}

	netAddr = &netAddress{
		ipaddr: "localhost",
		port:   8080,
	}
	sIntervalInt64 int64 = 300
)

func parseFlags() error {
	flag.Usage = usage
	flag.Var(netAddr, "a", "ip and port of server in format <ip>:<port>")
	flag.StringVar(&FlagsOptions.LogLevel, "l", "info", "loglevel")
	flag.Int64Var(&sIntervalInt64, "i", 300, "interval for metrics dump in seconds")
	flag.StringVar(&FlagsOptions.FileStoragePath, "f", "./metrics.dump", "Path to metrics dump file")
	flag.BoolVar(&FlagsOptions.Restore, "r", true, "load metrics from dump on start")
	flag.StringVar(&FlagsOptions.DatabaseDSN, "d", "", "Database DSN")
	flag.StringVar(&FlagsOptions.HashKey, "k", "", "Hash key")

	flag.Parse()

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		err := netAddr.Set(envRunAddr)
		if err != nil {
			return err
		}
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		FlagsOptions.LogLevel = envLogLevel
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		err := validateIntervalString(envStoreInterval)
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
		FlagsOptions.StoreInterval, err = time.ParseDuration(envStoreInterval + "s")
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
	} else {
		err := validateIntervalInt64(sIntervalInt64)
		if err != nil {
			return fmt.Errorf("flag -i: %w", err)
		}
		FlagsOptions.StoreInterval = time.Duration(sIntervalInt64) * time.Second
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		err := checkPathWritable(envFileStoragePath)
		if err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH value: %w", err)
		}
		FlagsOptions.FileStoragePath = envFileStoragePath
	} else {
		err := checkPathWritable(FlagsOptions.FileStoragePath)
		if err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH value: %w", err)
		}
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		var err error
		FlagsOptions.Restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			return fmt.Errorf("invalid RESTORE value: %w", err)
		}
	}

	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		FlagsOptions.DatabaseDSN = envDatabaseDSN
	}

	if envHashKey := os.Getenv("KEY"); envHashKey != "" {
		FlagsOptions.HashKey = envHashKey
	}
	return nil
}
