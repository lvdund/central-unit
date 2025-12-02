package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config groups all runtime configuration for the CU-CP binary.
type Config struct {
	CUCP     CUCPConfig     `yaml:"cucp"`
	F1AP     F1APConfig     `yaml:"f1ap"`
	E1AP     E1APConfig     `yaml:"e1ap"`
	NGAP     NGAPConfig     `yaml:"ngap"`
	Logging  LoggingConfig  `yaml:"logging"`
	Features FeatureFlags   `yaml:"features"`
	Tunables TunablesConfig `yaml:"tunables"`
}

type CUCPConfig struct {
	NodeID   string  `yaml:"node_id"`
	NodeName string  `yaml:"node_name"`
	PLMN     PLMN    `yaml:"plmn"`
	Slices   []Slice `yaml:"slices"`
	TAC      string  `yaml:"tac"`
}

type PLMN struct {
	MCC       string `yaml:"mcc"`
	MNC       string `yaml:"mnc"`
	MNCLength int    `yaml:"mnc_length"`
}

type Slice struct {
	SST string `yaml:"sst"`
	SD  string `yaml:"sd"`
}

type SCTPConfig struct {
	InStreams  uint16 `yaml:"in_streams"`
	OutStreams uint16 `yaml:"out_streams"`
}

type F1Timers struct {
	F1Setup time.Duration `yaml:"f1_setup_timer"`
}

type F1APConfig struct {
	LocalAddress string     `yaml:"local_address"`
	LocalPort    int        `yaml:"local_port"`
	SCTP         SCTPConfig `yaml:"sctp"`
	Timers       F1Timers   `yaml:"timers"`
}

type E1APConfig struct {
	LocalAddress string     `yaml:"local_address"`
	LocalPort    int        `yaml:"local_port"`
	SCTP         SCTPConfig `yaml:"sctp"`
}

type NGAPConfig struct {
	GnbId        string     `yaml:"gnb_id"`
	AMFAddress   string     `yaml:"amf_address"`
	AMFPort      int        `yaml:"amf_port"`
	LocalAddress string     `yaml:"local_address"`
	LocalPort    int        `yaml:"local_port"`
	SCTP         SCTPConfig `yaml:"sctp"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type FeatureFlags struct {
	SplitArchitecture      bool `yaml:"split_architecture"`
	ConnectedInactiveState bool `yaml:"connected_inactive"`
}

type TunablesConfig struct {
	UEStoreShards int `yaml:"ue_store_shards"`
}

// Load reads configuration from disk and applies defaults.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)

	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

// Validate ensures required fields are present and well-formed.
func (c Config) Validate() error {
	var problems []string

	if c.CUCP.NodeID == "" {
		problems = append(problems, "cucp.node_id must be non-zero")
	}
	if c.CUCP.NodeName == "" {
		problems = append(problems, "cucp.node_name is required")
	}
	if err := c.CUCP.PLMN.validate(); err != nil {
		problems = append(problems, fmt.Sprintf("cucp.plmn: %v", err))
	}

	if err := validateEndpoint("f1ap", c.F1AP.LocalAddress, c.F1AP.LocalPort); err != nil {
		problems = append(problems, err.Error())
	}
	if err := validateSCTP("f1ap.sctp", c.F1AP.SCTP); err != nil {
		problems = append(problems, err.Error())
	}
	if c.F1AP.Timers.F1Setup <= 0 {
		problems = append(problems, "f1ap.timers.f1_setup_timer must be >0")
	}

	if err := validateEndpoint("e1ap", c.E1AP.LocalAddress, c.E1AP.LocalPort); err != nil {
		problems = append(problems, err.Error())
	}
	if err := validateSCTP("e1ap.sctp", c.E1AP.SCTP); err != nil {
		problems = append(problems, err.Error())
	}

	if c.NGAP.AMFAddress == "" || c.NGAP.AMFPort <= 0 {
		problems = append(problems, "ngap.amf_address and ngap.amf_port are required")
	}
	if c.NGAP.LocalAddress == "" {
		problems = append(problems, "ngap.local_address is required")
	}
	if err := validateSCTP("ngap.sctp", c.NGAP.SCTP); err != nil {
		problems = append(problems, err.Error())
	}

	if c.Logging.Level == "" {
		problems = append(problems, "logging.level is required")
	}
	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		problems = append(problems, "logging.format must be either \"json\" or \"text\"")
	}

	if len(problems) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(problems, "; "))
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Tunables.UEStoreShards <= 0 {
		c.Tunables.UEStoreShards = 64
	}
}

func (p PLMN) validate() error {
	var problems []string
	if len(p.MCC) != 3 {
		problems = append(problems, "mcc must be 3 digits")
	}
	if p.MNCLength != 2 && p.MNCLength != 3 {
		problems = append(problems, "mnc_length must be 2 or 3")
	}
	if p.MNC == "" {
		problems = append(problems, "mnc is required")
	}
	if len(problems) > 0 {
		return fmt.Errorf(strings.Join(problems, "; "))
	}
	return nil
}

func validateEndpoint(name, addr string, port int) error {
	if addr == "" || port <= 0 {
		return fmt.Errorf("%s.local_address and %s.local_port must be set", name, name)
	}
	return nil
}

func validateSCTP(name string, cfg SCTPConfig) error {
	if cfg.InStreams == 0 || cfg.OutStreams == 0 {
		return fmt.Errorf("%s.in_streams and %s.out_streams must be non-zero", name, name)
	}
	return nil
}
