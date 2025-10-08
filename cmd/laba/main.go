package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/rx3lixir/laba/internal/config"
	//"github.com/rx3lixir/laba/internal/db"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      "2006-01-02 15:04:05",
		Level:           log.DebugLevel,
	})

	cm, err := config.NewConfigManager("internal/config/config.yaml")
	if err != nil {
		logger.Error("Error getting config file", "error", err)
		os.Exit(1)
	}

	c := cm.GetConfig()

	logger.Info("configuration", "user db link", c.UserDBParams.GetDSN())

	os.Exit(0)
}
