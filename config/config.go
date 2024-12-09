package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

// Target holds an IP and a range of ports to scan
type Target struct {
	IP               string   `yaml:"ip"`
	Name             string   `yaml:"name"`
	Range            string   `yaml:"range"`
	QueriesPerSecond int      `yaml:"queries_per_sec"`
	TCP              protocol `yaml:"tcp"`
	ICMP             protocol `yaml:"icmp"`
}

type protocol struct {
	Period   string `yaml:"period"`
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

// Conf holds configuration
type Conf struct {
	Timeout          int      `yaml:"timeout"`
	Limit            int      `yaml:"limit"`
	LogLevel         string   `yaml:"log_level"`
	QueriesPerSecond int      `yaml:"queries_per_sec"`
	TcpPeriod        string   `yaml:"tcp_period"`
	IcmpPeriod       string   `yaml:"icmp_period"`
	Targets          []Target `yaml:"targets"`
}

// New reads config from file and returns a config struct
func New(f string) (*Conf, error) {
	conf, err := os.Open(f)
	if err != nil {
		return nil, err
	}

	y, err := ioutil.ReadAll(conf)
	if err != nil {
		return nil, err
	}

	c := Conf{}

	if err = yaml.Unmarshal(y, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
