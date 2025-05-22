package main

import (
	"encoding/json"
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
}

func (f *Flags) FromRaw(raw *rawFlags) error {
	err := validation.ValidatePositiveString(raw.StoreInterval[:len(raw.StoreInterval)-1])
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
		raw.CryptoKey)
	return nil
}

func (f *Flags) SetN(netAddress NetAddress,
	logLevel string,
	storeInterval time.Duration,
	fileStoragePath string,
	restore bool,
	databaseDSN string,
	hashKey string,
	cryptoKey string) {
	f.NetAddress = netAddress
	f.LogLevel = logLevel
	f.StoreInterval = storeInterval
	f.FileStoragePath = fileStoragePath
	f.Restore = restore
	f.DatabaseDSN = databaseDSN
	f.HashKey = hashKey
	f.CryptoKey = cryptoKey
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
		err = validation.ValidatePositiveString(envStoreInterval)
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
		f.StoreInterval, err = time.ParseDuration(envStoreInterval + "s")
		if err != nil {
			return fmt.Errorf("invalid STORE_INTERVAL value: %w", err)
		}
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		err = validation.CheckPathWritable(envFileStoragePath)
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
		if validation.CheckFilePresence(envCryptoKey) {
			f.CryptoKey = envCryptoKey
		}
	}
	return nil
}

var (
	sIntervalInt64 int64  = 300
	configFile     string = ""

	cfgFromFile = rawFlags{
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
	}
	cli          Flags
	FlagsOptions Flags
)

func parseFlags() error {
	var err error
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
	flag.Parse()

	/*
		PRIORITY (least to most): CONFIG -> FLAGS -> ENV (if nothing given should be used default if possible)

		1. Check config present
		2. Read config file
		3. If config readed -> write it to FLAGS
		4. Parse flags
		5. if flag readed -> write it to FLAGS
		6. full validation of FlagOptions
		7. LoadENV
	*/

	if envConfig := os.Getenv("CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	if configFile != "" {
		if !validation.CheckFilePresence(configFile) {
			return fmt.Errorf("config file %q not found", configFile)
		}
		file, err := os.Open(configFile)
		if err != nil {
			return fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&cfgFromFile); err != nil {
			return fmt.Errorf("failed to decode config: %w", err)
		}
	}

	err = FlagsOptions.FromRaw(&cfgFromFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// flagOptions now config values or default values.
	// Next check if clioptions was given, and replace in FlagOptions them.
	if cli.NetAddress.isSet {
		FlagsOptions.NetAddress = cli.NetAddress
	}
	if cli.LogLevel != "" {
		FlagsOptions.LogLevel = cli.LogLevel
	}
	if sIntervalInt64 != 0 {
		err := validation.ValidatePositiveInt64(sIntervalInt64)
		if err != nil {
			return fmt.Errorf("flag -i: %w", err)
		}
		FlagsOptions.StoreInterval = time.Duration(sIntervalInt64) * time.Second
	}
	if cli.FileStoragePath != "" {
		err := validation.CheckPathWritable(cli.FileStoragePath)
		if err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH value: %w", err)
		}
		FlagsOptions.FileStoragePath = cli.FileStoragePath
	}
	if cli.Restore {
		FlagsOptions.Restore = cli.Restore
	}
	if cli.DatabaseDSN != "" {
		FlagsOptions.DatabaseDSN = cli.DatabaseDSN
	}
	if cli.HashKey != "" {
		FlagsOptions.HashKey = cli.HashKey
	}
	if cli.CryptoKey != "" {
		if validation.CheckFilePresence(cli.CryptoKey) {
			FlagsOptions.CryptoKey = cli.CryptoKey
		}
	}

	/// Now all defaults/config values was overridden by command line options
	// Next check env and finally validate

	err = FlagsOptions.LoadENV()
	if err != nil {
		return fmt.Errorf("failed to load ENV flags: %w", err)
	}

	if !validation.CheckFilePresence(FlagsOptions.CryptoKey) {
		FlagsOptions.CryptoKey, err = validation.FindKEYFile()
		if err != nil {
			return fmt.Errorf("invalid CRYPTO_KEY value: file '%s' does not exists", FlagsOptions.CryptoKey)
		}
	}

	return nil
}
