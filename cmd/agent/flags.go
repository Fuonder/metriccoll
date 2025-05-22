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
	ErrNotFullIP       = errors.New("given ip address and port incorrect")
	ErrInvalidIP       = errors.New("incorrect ip address")
	ErrInvalidPort     = errors.New("incorrect port number")
	ErrInvalidArgument = errors.New("invalid argument")
)

var (
	version  = "0.1.21"
	progName = "Fuonder's ya-practicum client"
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

type NetAddress struct {
	IPAddr string
	Port   int
	isSet  bool
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

type rawCliOptions struct {
	NetAddr        NetAddress `json:"address"`
	ReportInterval string     `json:"report_interval"`
	PollInterval   string     `json:"poll_interval"`
	HashKey        string     `json:"hash_key"`
	RateLimit      int64      `json:"rate_limit"`
	CryptoKey      string     `json:"crypto_key"`
}

type CliOptions struct {
	NetAddr        NetAddress    `json:"address"`
	ReportInterval time.Duration `json:"report_interval"`
	PollInterval   time.Duration `json:"poll_interval"`
	HashKey        string        `json:"hash_key"`
	RateLimit      int64         `json:"rate_limit"`
	CryptoKey      string        `json:"crypto_key"`
}

func (o *CliOptions) String() string {
	return fmt.Sprintf(
		"netAddr:%s, "+
			"reportInterval:%s, "+
			"pollInterval:%s, "+
			"hashKey:%s, "+
			"rateLimit: %d, "+
			"CryptoKey: %s",
		o.NetAddr.String(),
		o.ReportInterval,
		o.PollInterval,
		o.HashKey,
		o.RateLimit,
		o.CryptoKey,
	)
}
func (o *CliOptions) FromRaw(raw *rawCliOptions) error {
	var err error

	err = validation.ValidateNonNegativeString(raw.ReportInterval[:len(raw.ReportInterval)-1])
	if err != nil {
		return err
	}
	rt, err := time.ParseDuration(raw.ReportInterval)
	if err != nil {
		return err
	}

	err = validation.ValidateNonNegativeString(raw.PollInterval[:len(raw.PollInterval)-1])
	if err != nil {
		return err
	}
	pt, err := time.ParseDuration(raw.PollInterval)
	if err != nil {
		return err
	}

	err = validation.ValidateNonNegativeInt64(raw.RateLimit)
	if err != nil {
		return err
	}

	o.SetN(raw.NetAddr, rt, pt, raw.HashKey, raw.RateLimit, raw.CryptoKey)
	return nil
}

func (o *CliOptions) SetN(netAddress NetAddress,
	reportInterval time.Duration,
	pollInterval time.Duration,
	hashKey string,
	rateLimit int64,
	cryptoKey string) {
	o.NetAddr = netAddress
	o.ReportInterval = reportInterval
	o.PollInterval = pollInterval
	o.HashKey = hashKey
	o.RateLimit = rateLimit
	o.CryptoKey = cryptoKey
}

func (o *CliOptions) Copy(another *CliOptions) {
	o.NetAddr = another.NetAddr
	o.ReportInterval = another.ReportInterval
	o.PollInterval = another.PollInterval
	o.HashKey = another.HashKey
	o.RateLimit = another.RateLimit
	o.CryptoKey = another.CryptoKey
}

func (o *CliOptions) LoadENV() error {
	var err error

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		err = o.NetAddr.Set(envRunAddr)
		if err != nil {
			return err
		}
	}

	if envRInterval := os.Getenv("REPORT_INTERVAL"); envRInterval != "" {
		err = validation.ValidateNonNegativeString(envRInterval)
		if err != nil {
			return fmt.Errorf("REPORT_INTERVAL: %w", err)
		}
		o.ReportInterval, err = time.ParseDuration(envRInterval + "s")
		if err != nil {
			return fmt.Errorf("REPORT_INTERVAL: %w", err)
		}
	}

	if envPInterval := os.Getenv("POLL_INTERVAL"); envPInterval != "" {
		err = validation.ValidateNonNegativeString(envPInterval)
		if err != nil {
			return fmt.Errorf("POLL_INTERVAL: %w", err)
		}
		o.PollInterval, err = time.ParseDuration(envPInterval + "s")
		if err != nil {
			return fmt.Errorf("POLL_INTERVAL: %w", err)
		}
	}

	if envHashKey := os.Getenv("KEY"); envHashKey != "" {
		o.HashKey = envHashKey
	}

	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		err = validation.ValidateNonNegativeString(envRateLimit)
		if err != nil {
			return fmt.Errorf("RATE_LIMIT: %w", err)
		}
		o.RateLimit, err = strconv.ParseInt(envRateLimit, 10, 64)
		if err != nil {
			return fmt.Errorf("RATE_LIMIT: %w", err)
		}
	}

	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		if validation.CheckFilePresence(envCryptoKey) {
			o.CryptoKey = envCryptoKey
		}
	}
	return nil
}

var (
	cfgFromFile = rawCliOptions{
		NetAddr: NetAddress{
			IPAddr: "localhost",
			Port:   8080},
		ReportInterval: "10s",
		PollInterval:   "2s",
		HashKey:        "",
		RateLimit:      1,
		CryptoKey:      "./certs/server.crt",
	}
	pInterval  int64  = 2
	rInterval  int64  = 10
	configFile string = ""

	CliOpt CliOptions
	cli    CliOptions
)

func parseFlags() error {
	var err error

	flag.Usage = usage
	flag.Var(&cli.NetAddr, "a", "ip and port of server in format <ip>:<port>")
	flag.Int64Var(&pInterval, "p", 0, "interval of collecting metrics in secs")
	flag.Int64Var(&rInterval, "r", 0, "interval of reports in secs")
	flag.StringVar(&cli.HashKey, "k", "", "key for hash")
	flag.Int64Var(&cli.RateLimit, "l", 0, "rate limit")
	flag.StringVar(&cli.CryptoKey, "crypto-key", "", "Path to private key file")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	flag.StringVar(&configFile, "c", "", "Path to config file")
	flag.Parse()

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

	err = CliOpt.FromRaw(&cfgFromFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cli.NetAddr.isSet {
		CliOpt.NetAddr = cli.NetAddr
	}

	if pInterval != 0 {
		err = validation.ValidateNonNegativeInt64(pInterval)
		if err != nil {
			return fmt.Errorf("flag -p: %w", err)
		}
		CliOpt.PollInterval = time.Duration(pInterval) * time.Second
	}

	if rInterval != 0 {
		err = validation.ValidateNonNegativeInt64(rInterval)
		if err != nil {
			return fmt.Errorf("flag -r: %w", err)
		}
		CliOpt.ReportInterval = time.Duration(rInterval) * time.Second
	}

	if cli.HashKey != "" {
		CliOpt.HashKey = cli.HashKey
	}

	if cli.RateLimit != 0 {
		err = validation.ValidateNonNegativeInt64(CliOpt.RateLimit)
		if err != nil {
			return fmt.Errorf("flag -l: %w", err)
		}
		CliOpt.RateLimit = cli.RateLimit
	}

	if cli.CryptoKey != "" {
		CliOpt.CryptoKey = cli.CryptoKey
	}

	err = CliOpt.LoadENV()
	if err != nil {
		return fmt.Errorf("failed to load ENV flags: %w", err)
	}

	if !validation.CheckFilePresence(CliOpt.CryptoKey) {
		CliOpt.CryptoKey, err = validation.FindCRTFile()
		if err != nil {
			return fmt.Errorf("invalid CRYPTO_KEY value: file '%v' does not exists", CliOpt.CryptoKey)
		}
	}

	return nil
}
