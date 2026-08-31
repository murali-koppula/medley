package cmd

const appName = "medley"
const commandVersion = "0.1.0"
const cookieFileBasename = "cookies.txt"

// TASK_TIMEOUT_MINUTES must be >= sum of (
//     YT_DLP_TIMEOUT_MINUTES
//     FFMPEG_TIMEOUT_MINUTES
//     CONVERT_TIMEOUT_SEC
//     EYED3_TIMEOUT_SEC
//     ATOMICPARSLEY_TIMEOUT_SEC
// )
const TASK_TIMEOUT_MINUTES = 5
const YT_DLP_TIMEOUT_MINUTES = 5
const FFMPEG_TIMEOUT_MINUTES = 5
const CONVERT_TIMEOUT_SEC = 60
const EYED3_TIMEOUT_SEC = 60
const ATOMICPARSLEY_TIMEOUT_SEC = 60

// Pause processing due to youtube rate limits.
const YT_BASE_TRACK_PAUSE_SEC = 2  // Base pause between each track
const YT_BATCH_PAUSE_INTERVAL = 10 // Take a longer break every N tracks
const YT_BATCH_PAUSE_SEC = 12      // Duration of the longer break in seconds

const WG_CLEANUP_TIMEOUT_SEC = 60

type Tokens struct {
	mediaConfigFileName string
	mediaHome           string
	verbose             bool
}

var tokens = &Tokens{}
var logLevels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "OFF"}
