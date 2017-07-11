package config

import (
	"github.com/cloudfoundry-incubator/candiedyaml"
	vcap "github.com/dinp/gorouter/common"

	"io/ioutil"
	"time"
)

type StatusConfig struct {
	Port uint16 `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

var defaultStatusConfig = StatusConfig{
	Port: 8082,
	User: "",
	Pass: "",
}

type LoggingConfig struct {
	File               string `yaml:"file"`
	Syslog             string `yaml:"syslog"`
	Level              string `yaml:"level"`
	LoggregatorEnabled bool   `yaml:"loggregator_enabled"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: "debug",
}

type Config struct {
	Status  StatusConfig  `yaml:"status"`
	Logging LoggingConfig `yaml:"logging"`

	Port        uint16 `yaml:"port"`
	Index       uint   `yaml:"index"`
	GoMaxProcs  int    `yaml:"go_max_procs,omitempty"`
	TraceKey    string `yaml:"trace_key"`
	RedisServer string `yaml:"redis_server"`
	AccessLog   string `yaml:"access_log"`

	ReloadUriIntervalInSeconds int `yaml:"reload_uri_interval"`
	EndpointTimeoutInSeconds   int `yaml:"endpoint_timeout"`
	DrainTimeoutInSeconds      int `yaml:"drain_timeout,omitempty"`

	// These fields are populated by the `Process` function.
	ReloadUriInterval time.Duration `yaml:"-"`
	EndpointTimeout   time.Duration `yaml:"-"`
	DrainTimeout      time.Duration `yaml:"-"`

	Ip string `yaml:"-"`
}

var defaultConfig = Config{
	Status:  defaultStatusConfig,
	Logging: defaultLoggingConfig,

	Port:        8081,
	Index:       0,
	GoMaxProcs:  8,
	RedisServer: "127.0.0.1:6379",

	EndpointTimeoutInSeconds: 60,

	ReloadUriIntervalInSeconds: 5,
}

func DefaultConfig() *Config {
	c := defaultConfig

	c.Process()

	return &c
}

func (c *Config) Process() {
	var err error

	c.ReloadUriInterval = time.Duration(c.ReloadUriIntervalInSeconds) * time.Second
	c.EndpointTimeout = time.Duration(c.EndpointTimeoutInSeconds) * time.Second

	drain := c.DrainTimeoutInSeconds
	if drain == 0 {
		drain = c.EndpointTimeoutInSeconds
	}
	c.DrainTimeout = time.Duration(drain) * time.Second

	c.Ip, err = vcap.LocalIP()
	if err != nil {
		panic(err)
	}
}

func (c *Config) Initialize(configYAML []byte) error {
	return candiedyaml.Unmarshal(configYAML, &c)
}

func InitConfigFromFile(path string) *Config {
	var c *Config = DefaultConfig()
	var e error

	b, e := ioutil.ReadFile(path)
	if e != nil {
		panic(e.Error())
	}

	e = c.Initialize(b)
	if e != nil {
		panic(e.Error())
	}

	c.Process()

	return c
}
