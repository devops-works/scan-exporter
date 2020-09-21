package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// target holds an IP and a range of ports to scan
type target struct {
	Name string   `yaml:"name"`
	IP   string   `yaml:"ip"`
	TCP  protocol `yaml:"tcp"`
	UDP  protocol `yaml:"udp"`
}

type protocol struct {
	Period   string `yaml:"period"`
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

// Conf holds configuration
type Conf struct {
	Targets []target `yaml:"targets"`
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
