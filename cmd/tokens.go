package cmd

type Tokens struct {
	mediaConfigFileName string
	mediaHome           string
	verbose             bool
}

const commandVersion = "0.1.0"

var tokens = &Tokens{}
var logLevels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "OFF"}

const WG_CLEANUP_TIMEOUT_SEC = 60

// TASK_TIMEOUT_MINUTES must be >= sum of (
//     YT_DLP_TIMEOUT_MINUTES
//     FFMPEG_TIMEOUT_MINUTES
//     CONVERT_TIMEOUT_SEC
//     EYED3_TIMEOUT_SEC
//     ATOMICPARSLEY_TIMEOUT_SEC
// )
const TASK_TIMEOUT_MINUTES = 5
