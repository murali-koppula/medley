package cmd

import (
	logging "gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger(appName)

func GetLogger() *logging.Logger {
	return log
}
