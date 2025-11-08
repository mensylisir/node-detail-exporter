package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ScrapeInterval time.Duration `yaml:"scrape_interval"`
	Collectors     struct {
		Pidstat   bool `yaml:"pidstat"`
		Iostat    bool `yaml:"iostat"`
		Vmstat    bool `yaml:"vmstat"`
		DiskUsage bool `yaml:"diskusage"`
		Benchmark bool `yaml:"benchmark"`
		Ping      struct {
			Enabled bool     `yaml:"enabled"`
			Targets []string `yaml:"targets"`
		} `yaml:"ping"`
		Audit struct {
			Enabled bool   `yaml:"enabled"`
			LogPath string `yaml:"log_path"`
		} `yaml:"audit"`
		Mtr struct {
			Enabled bool     `yaml:"enabled"`
			Targets []string `yaml:"targets"`
		} `yaml:"mtr"`
	} `yaml:"collectors"`
}

func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
