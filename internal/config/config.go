package config

import (
	"errors"
	"fmt"
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

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.ScrapeInterval <= 0 {
		return errors.New("scrape_interval must be a positive duration")
	}

	collectorsEnabled := c.Collectors.Pidstat ||
		c.Collectors.Iostat ||
		c.Collectors.Vmstat ||
		c.Collectors.DiskUsage ||
		c.Collectors.Benchmark ||
		c.Collectors.Ping.Enabled ||
		c.Collectors.Audit.Enabled ||
		c.Collectors.Mtr.Enabled
	if !collectorsEnabled {
		return errors.New("at least one collector must be enabled")
	}

	if c.Collectors.Ping.Enabled && len(c.Collectors.Ping.Targets) == 0 {
		return errors.New("ping collector is enabled but has no targets")
	}

	if c.Collectors.Mtr.Enabled && len(c.Collectors.Mtr.Targets) == 0 {
		return errors.New("mtr collector is enabled but has no targets")
	}

	if c.Collectors.Audit.Enabled && c.Collectors.Audit.LogPath == "" {
		return errors.New("audit collector is enabled but log_path is not specified")
	}

	return nil
}
