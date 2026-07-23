package cmd

type Tokens struct {
	mediaConfigFileName string
	mediaHome           string
	verbose             bool
}

const commandVersion = "0.1.0"

var tokens = &Tokens{}
var logLevels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "OFF"}
