package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/validation/filevalidation"
	"github.com/Fuonder/metriccoll.git/internal/validation/numericvalidation"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	version  = "0.1.25"
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

type NetAddress struct {
	ipaddr string
	port   int
	isSet  bool
}

func (n *NetAddress) String() string {
	return fmt.Sprintf("%s:%d", n.ipaddr, n.port)
}

func (n *NetAddress) Set(value string) error {
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
	n.isSet = true
	return nil
}
func (n *NetAddress) UnmarshalJSON(data []byte) error {
	var addr string
	if err := json.Unmarshal(data, &addr); err != nil {
		return err
	}
	return n.Set(addr)
}

type rawFlags struct {
	NetAddress      NetAddress `json:"address"`
	LogLevel        string     `json:"log_level,omitempty"`
	StoreInterval   string     `json:"store_interval"`
	FileStoragePath string     `json:"store_file"`
	Restore         bool       `json:"restore"`
	DatabaseDSN     string     `json:"database_dsn"`
	HashKey         string     `json:"hash_key,omitempty"`
	CryptoKey       string     `json:"crypto_key"`
	TrustedSubnet   string     `json:"trusted_subnet"`
	GRPCAddress     string     `json:"grpc_address"`
}

type Flags struct {
	NetAddress      NetAddress    `json:"address"`
	LogLevel        string        `json:"log_level,omitempty"`
	StoreInterval   time.Duration `json:"store_interval"`
	FileStoragePath string        `json:"store_file"`
	Restore         bool          `json:"restore"`
	DatabaseDSN     string        `json:"database_dsn"`
	HashKey         string        `json:"hash_key,omitempty"`
	CryptoKey       string        `json:"crypto_key"`
	TrustedSubnet   string        `json:"trusted_subnet"`
	GRPCAddress     string        `json:"grpc_address"`
}

func (f *Flags) ReadArgv(cli Flags, sInt int64) error {
	if cli.NetAddress.isSet {
		f.NetAddress = cli.NetAddress
	}
	if cli.LogLevel != "" {
		f.LogLevel = cli.LogLevel
	}
	if sInt != 0 {
		err := numericvalidation.ValidatePositiveInt64(sInt)
		if err != nil {
			return fmt.Errorf("flag -i: %w", err)
		}
		f.StoreInterval = time.Duration(sInt) * time.Second
	}
	if cli.FileStoragePath != "" {
		err := filevalidation.CheckPathWritable(cli.FileStoragePath)
		if err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH value: %w", err)
		}
		f.FileStoragePath = cli.FileStoragePath
	}
	if cli.Restore {
		f.Restore = cli.Restore
	}
	if cli.DatabaseDSN != "" {
		f.DatabaseDSN = cli.DatabaseDSN
	}
	if cli.HashKey != "" {
		f.HashKey = cli.HashKey
	}
	if cli.CryptoKey != "" {
		if filevalidation.CheckFilePresence(cli.CryptoKey) {
			f.CryptoKey = cli.CryptoKey
		}
	}
	if cli.TrustedSubnet != "" {
		f.TrustedSubnet = cli.TrustedSubnet
	}
	if cli.GRPCAddress != "" {
		f.GRPCAddress = cli.GRPCAddress
	}
	return nil
}

func (f *Flags) ReadConfig(from string) error {
	var cfgFromFile = rawFlags{
		NetAddress: NetAddress{
			ipaddr: "localhost",
			port:   8080},
		LogLevel:        "debug",
		StoreInterval:   "300s",
		FileStoragePath: "./metrics.dump",
		Restore:         true,
		DatabaseDSN:     "postgres://videos:12345678@localhost:5432/videos?sslmode=disable",
		HashKey:         "",
		CryptoKey:       "./certs/server.key",
		TrustedSubnet:   "",
		GRPCAddress:     ":3333",
	}
	if from != "" {
		if !filevalidation.CheckFilePresence(from) {
			return fmt.Errorf("config file %q not found", from)
		}
		file, err := os.Open(from)
		if err != nil {
			return fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&cfgFromFile); err != nil {
			return fmt.Errorf("failed to decode config: %w", err)
		}
	}

	err := f.FromRaw(&cfgFromFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

func (f *Flags) FromRaw(raw *rawFlags) error {
	err := numericvalidation.ValidatePositiveString(raw.StoreInterval[:len(raw.StoreInterval)-1])
	if err != nil {
		return fmt.Errorf("invalid StoreInterval value: %w", err)
	}
	t, err := time.ParseDuration(raw.StoreInterval)
	if err != nil {
		return fmt.Errorf("invalid StoreInterval value: %w", err)
	}

	f.SetN(raw.NetAddress,
		raw.LogLevel,
		t,
		raw.FileStoragePath,
		raw.Restore,
		raw.DatabaseDSN,
		raw.HashKey,
		raw.CryptoKey,
		raw.TrustedSubnet,
		raw.GRPCAddress)
	return nil
}

func (f *Flags) SetN(netAddress NetAddress,
	logLevel string,
	storeInterval time.Duration,
	fileStoragePath string,
	restore bool,
	databaseDSN string,
	hashKey string,
	cryptoKey string,
	trustedSubnet string,
	GRPCAddress string) {
	f.NetAddress = netAddress
	f.LogLevel = logLevel
	f.StoreInterval = storeInterval
	f.FileStoragePath = fileStoragePath
	f.Restore = restore
	f.DatabaseDSN = databaseDSN
	f.HashKey = hashKey
	f.CryptoKey = cryptoKey
	f.TrustedSubnet = trustedSubnet
	f.GRPCAddress = GRPCAddress
}

func (f *Flags) Copy(another *Flags) {
	f.NetAddress = another.NetAddress
	f.LogLevel = another.LogLevel
	f.StoreInterval = another.StoreInterval
	f.FileStoragePath = another.FileStoragePath
	f.Restore = another.Restore
	f.DatabaseDSN = another.DatabaseDSN
	f.HashKey = another.HashKey
	f.CryptoKey = another.CryptoKey
	f.TrustedSubnet = another.TrustedSubnet
	f.GRPCAddress = another.GRPCAddress
}

func (f *Flags) String() string {
	return fmt.Sprintf("netAddr: %s, "+
		"LogLevel: %s, "+
		"StoreInterval: %s, "+
		"FileStoragePath: %s, "+
		"Restore: %v, "+
		"DatabaseDSN: %s, "+
		"HashKey: %s, "+
		"CryptoKey: %s "+
		"TrustedSubnet: %s "+
		"GRPCAddress: %s",
		f.NetAddress.String(),
		f.LogLevel,
		f.StoreInterval.String(),
		f.FileStoragePath,
		f.Restore,
		f.DatabaseDSN,
		f.HashKey,
		f.CryptoKey,
		f.TrustedSubnet,
		f.GRPCAddress,
	)
}

func (f *Flags) LoadENV() error {
	/*
		env list =
		ADDRESS -> NetAddres
		LOG_LEVEL -> LogLevel
		STORE_INTERVAL -> StoreInterval
		FILE_STORAGE_PATH -> FileStoragePath
		RESTORE -> Restore
		DATABASE_DSN -> DatabaseDSN
		KEY -> HashKey
		CRYPTO_KEY -> CryptoKey
	*/

	var err error

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		err = f.NetAddress.Set(envRunAddr)
		if err != nil {
			return err
		}
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		f.LogLevel = envLogLevel
	}

	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		err = numericvalidation.ValidatePositiveString(envStoreInterval)
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
		f.StoreInterval, err = time.ParseDuration(envStoreInterval + "s")
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		err = filevalidation.CheckPathWritable(envFileStoragePath)
		if err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH value: %w", err)
		}
		f.FileStoragePath = envFileStoragePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		f.Restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			return fmt.Errorf("invalid RESTORE value: %w", err)
		}
	}

	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		f.DatabaseDSN = envDatabaseDSN
	}

	if envHashKey := os.Getenv("KEY"); envHashKey != "" {
		f.HashKey = envHashKey
	}

	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		if filevalidation.CheckFilePresence(envCryptoKey) {
			f.CryptoKey = envCryptoKey
		}
	}
	if envTrustedSubnet := os.Getenv("TRUSTED_SUBNET"); envTrustedSubnet != "" {
		f.TrustedSubnet = envTrustedSubnet
	}
	if envGRPCAddress := os.Getenv("GRPC_ADDRESS"); envGRPCAddress != "" {
		f.GRPCAddress = envGRPCAddress
	}
	return nil
}

var FlagsOptions Flags

func parseFlags() error {
	var (
		err            error
		sIntervalInt64 int64  = 300
		configFile     string = ""
		cli            Flags
	)
	flag.Usage = usage
	flag.Var(&cli.NetAddress, "a", "ip and port of server in format <ip>:<port>")
	flag.StringVar(&cli.LogLevel, "l", "", "loglevel")
	flag.Int64Var(&sIntervalInt64, "i", 0, "interval for metrics dump in seconds")
	flag.StringVar(&cli.FileStoragePath, "f", "", "Path to metrics dump file")
	flag.BoolVar(&cli.Restore, "r", false, "load metrics from dump on start")
	flag.StringVar(&cli.DatabaseDSN, "d", "", "Database DSN")
	flag.StringVar(&cli.HashKey, "k", "", "Hash key")
	flag.StringVar(&cli.CryptoKey, "crypto-key", "", "Path to private key file")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	flag.StringVar(&configFile, "c", "", "Path to config file")
	flag.StringVar(&cli.TrustedSubnet, "t", "", "Trusted subnet for incoming requests")
	flag.StringVar(&cli.GRPCAddress, "g", "", "ip and port for GRPC service")
	flag.Parse()

	if envConfig := os.Getenv("CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	err = FlagsOptions.ReadConfig(configFile)
	if err != nil {
		return err
	}

	err = FlagsOptions.ReadArgv(cli, sIntervalInt64)
	if err != nil {
		return err
	}

	err = FlagsOptions.LoadENV()
	if err != nil {
		return fmt.Errorf("failed to load ENV flags: %w", err)
	}

	if !filevalidation.CheckFilePresence(FlagsOptions.CryptoKey) {
		FlagsOptions.CryptoKey, err = filevalidation.FindKEYFile()
		if err != nil {
			return fmt.Errorf("invalid CRYPTO_KEY value: file '%s' does not exists", FlagsOptions.CryptoKey)
		}
	}

	return nil
}
