package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/validation"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	version  = "0.1.21"
	progName = "Fuonder's ya-practicum server"
	source   = "https://github.com/Fuonder/metriccoll"
)

var usage = func() {
	_, err := fmt.Fprintf(flag.CommandLine.Output(), "%s\nSource code:\t%s\nVersion:\t%s\nUsage of %s:\n",
		progName,
		source,
		version,
		progName)
	if err != nil {
		return
	}
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
	CryptoKey       string
}

func (f *Flags) String() string {
	return fmt.Sprintf("netAddr: %s, "+
		"LogLevel: %s, "+
		"StoreInterval: %s, "+
		"FileStoragePath: %s, "+
		"Restore: %v, "+
		"DatabaseDSN: %s, "+
		"HashKey: %s, "+
		"CryptoKey: %s",
		f.NetAddress.String(),
		f.LogLevel,
		f.StoreInterval.String(),
		f.FileStoragePath,
		f.Restore,
		f.DatabaseDSN,
		f.HashKey,
		f.CryptoKey,
	)
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
		CryptoKey:       "./certs/server.key",
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
	flag.StringVar(&FlagsOptions.CryptoKey, "crypto-key", "./certs/server.key", "Path to private key file")

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
		err := validation.ValidatePositiveString(envStoreInterval)
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
		FlagsOptions.StoreInterval, err = time.ParseDuration(envStoreInterval + "s")
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
	} else {
		err := validation.ValidatePositiveInt64(sIntervalInt64)
		if err != nil {
			return fmt.Errorf("flag -i: %w", err)
		}
		FlagsOptions.StoreInterval = time.Duration(sIntervalInt64) * time.Second
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		err := validation.CheckPathWritable(envFileStoragePath)
		if err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH value: %w", err)
		}
		FlagsOptions.FileStoragePath = envFileStoragePath
	} else {
		err := validation.CheckPathWritable(FlagsOptions.FileStoragePath)
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

	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		if validation.CheckFilePresence(envCryptoKey) {
			FlagsOptions.CryptoKey = envCryptoKey
		}
	} else {
		if !validation.CheckFilePresence(FlagsOptions.CryptoKey) {
			var err error
			FlagsOptions.CryptoKey, err = validation.FindKEYFile()
			if err != nil {
				return fmt.Errorf("invalid CRYPTO_KEY value: file '%s' does not exists\n curdir:", FlagsOptions.CryptoKey)
			}
		}
	}

	return nil
}
