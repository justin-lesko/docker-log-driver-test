package main

import (
	"fmt"
	"os"

	"github.com/cpuguy83/docker-log-driver-test/internal/http"
	"github.com/cpuguy83/docker-log-driver-test/internal/logger"
	"github.com/docker/go-plugins-helpers/sdk"
	"github.com/sirupsen/logrus"
)

var logLevels = map[string]logrus.Level{
	"debug": logrus.DebugLevel,
	"info":  logrus.InfoLevel,
	"warn":  logrus.WarnLevel,
	"error": logrus.ErrorLevel,
}

func main() {
	levelVal := os.Getenv("LOG_LEVEL")
	if levelVal == "" {
		levelVal = "info"
	}
	if level, exists := logLevels[levelVal]; exists {
		logrus.SetLevel(level)
	} else {
		fmt.Fprintln(os.Stderr, "invalid log level: ", levelVal)
		os.Exit(1)
	}

	h := sdk.NewHandler(`{"Implements": ["LogDriver"]}`)
	http.Handlers(&h, logger.NewLogger())
	if err := h.ServeUnix("datadog", 0); err != nil {
		panic(err)
	}
}
