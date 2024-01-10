package config

import (
	"github.com/spf13/viper"
	"log"
	"log/slog"
	"os"
	"strings"
)

func init() {
	// Find home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	// Search config in home directory with name ".docker-profiler-go" (without extension).
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("application")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("hello")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "hello-world"))
}

// GetMetricsEndpoint returns metrics collector endpoint
func GetMetricsEndpoint() string {
	return viper.GetString("telemetry.metrics.endpoint")
}

// GetTracesEndpoint returns trace collector endpoint
func GetTracesEndpoint() string {
	return viper.GetString("telemetry.traces.endpoint")
}
