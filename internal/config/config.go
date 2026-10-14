// Package config centralizes configuration loading.
// Precedence (highest first): flags > env vars > config file > defaults.
//
// Env vars use the RT_ prefix:
//
//	RT_DB_PATH, RT_LOG_LEVEL, RT_LOG_FORMAT, RT_LOG_SOURCE,
//	RT_METRICS_ADDR, RT_DEFAULT_TIMEOUT
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	DBPath         string        `mapstructure:"db-path"`
	LogLevel       string        `mapstructure:"log-level"`
	LogFormat      string        `mapstructure:"log-format"`
	LogSource      bool          `mapstructure:"log-source"`
	MetricsAddr    string        `mapstructure:"metrics-addr"`
	DefaultTimeout time.Duration `mapstructure:"default-timeout"`
}

// Load reads and merges configuration sources for the currently-executing
// command. Call this from PersistentPreRunE once flags are parsed.
func Load(cmd *cobra.Command) (Config, error) {
	v := viper.New()

	// --- Config file ---
	// If --config was provided, use it. Otherwise look in the OS config dir.
	if cfgFile, _ := cmd.Flags().GetString("config"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		if dir, err := os.UserConfigDir(); err == nil {
			v.AddConfigPath(filepath.Join(dir, "reading-tracker"))
		}
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// --- Environment ---
	// RT_LOG_LEVEL  →  key "log-level"
	v.SetEnvPrefix("RT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	// --- Defaults ---
	v.SetDefault("db-path", defaultDBPath())
	v.SetDefault("log-level", "info")
	v.SetDefault("log-format", "text")
	v.SetDefault("log-source", false)
	v.SetDefault("metrics-addr", "")
	v.SetDefault("default-timeout", 2*time.Second)

	// --- Config file (optional) ---
	if err := v.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if !errors.As(err, &nf) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		// Config file absent is fine — defaults + env + flags suffice.
	}

	// --- Flags (highest priority) ---
	// Bind root persistent flags and the executing command's flags.
	// Viper reads flag values at Unmarshal time, so this picks up what the
	// user actually passed.
	if err := v.BindPFlags(cmd.Root().PersistentFlags()); err != nil {
		return Config{}, fmt.Errorf("bind persistent flags: %w", err)
	}
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return Config{}, fmt.Errorf("bind flags: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}

// ConfigFilePath returns the path Viper would use by default.
// Exposed so `reading-tracker version` can display it.
func ConfigFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(dir, "reading-tracker", "config.yaml")
}

func defaultDBPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "reading.db"
	}
	return filepath.Join(dir, "reading-tracker", "reading.db")
}
