package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type SSHConfig struct {
	ConnectTimeout int    `mapstructure:"connect_timeout"`
	CommandTimeout int    `mapstructure:"command_timeout"`
	IdentityFile   string `mapstructure:"identity_file"`
	User           string `mapstructure:"user"`
}

type Config struct {
	Nodes     []string            `mapstructure:"nodes"`
	Interval  float64             `mapstructure:"interval"`
	Workers   int                 `mapstructure:"workers"`
	View      string              `mapstructure:"view"`
	Processes bool                `mapstructure:"processes"`
	Debug     bool                `mapstructure:"debug"`
	Static    bool                `mapstructure:"static"`
	Compact   bool                `mapstructure:"compact"`
	Group     string              `mapstructure:"group"`
	SSH       SSHConfig           `mapstructure:"ssh"`
	Groups    map[string][]string `mapstructure:"groups"`
}

func Defaults() Config {
	return Config{
		Interval: 2.0,
		Workers:  8,
		View:     "panel",
		SSH: SSHConfig{
			ConnectTimeout: 5,
			CommandTimeout: 10,
		},
		Groups: make(map[string][]string),
	}
}

func LoadFromFile(path string) (Config, error) {
	cfg := Defaults()

	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return cfg, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}

	return cfg, nil
}

// Load tries to load from the default config path. Returns defaults if no file found.
func Load() Config {
	cfg := Defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	path := filepath.Join(home, ".config", "node-monitor", "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		return cfg
	}
	return loaded
}

// ResolveNodes returns the node list based on group selection.
func (c Config) ResolveNodes(group string) []string {
	if group != "" {
		if nodes, ok := c.Groups[group]; ok {
			return nodes
		}
	}
	return c.Nodes
}
