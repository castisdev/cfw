package main

import (
	"fmt"
	"net"
	"os"

	"github.com/castisdev/cilog"

	"github.com/spf13/viper"
)

// Config :
type Config struct {
	IFName                   string `mapstructure:"if_name"`
	StorageUsageLimitPercent uint   `mapstructure:"storage_usage_limit_percent"`
	LogDir                   string `mapstructure:"log_dir"`
	LogLevel                 string `mapstructure:"log_level"`
	ListenAddr               string `mapstructure:"listen_addr"`
	CFMAddr                  string `mapstructure:"cfm_addr"`
	DownloaderBin            string `mapstructure:"downloader_bin"`
	BaseDir                  string `mapstructure:"base_dir"`
	DownloaderSleepSec       uint   `mapstructure:"downloader_sleep_sec"`
}

// ReadConfig :
func ReadConfig(configFile string) (*Config, error) {
	viper.SetDefault("downloader_sleep_sec", uint(5))
	viper.SetDefault("storage_usage_limit_percent", uint(90))

	var c Config
	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		return &Config{}, err
	}

	if err := viper.Unmarshal(&c); err != nil {
		return &Config{}, err
	}

	return &c, nil
}

// ValidationConfig :
func ValidationConfig(c Config) {

	if _, err := os.Stat(c.DownloaderBin); os.IsNotExist(err) {
		fmt.Printf("not exists file (%s)\n", err)
		os.Exit(-1)
	}

	if _, err := os.Stat(c.BaseDir); os.IsNotExist(err) {
		fmt.Printf("not exists dir (%s)\n", err)
		os.Exit(-1)
	}

	if _, err := cilog.LevelFromString(c.LogLevel); err != nil {
		fmt.Printf("invalid log level : error(%s)", err)
		os.Exit(-1)
	}

	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		fmt.Printf("invalid listen_addr : error(%s)", err)
		os.Exit(-1)
	}

	if _, _, err := net.SplitHostPort(c.CFMAddr); err != nil {
		fmt.Printf("invalid takser_addr : error(%s)", err)
		os.Exit(-1)
	}
}
