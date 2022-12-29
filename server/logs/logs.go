package logs

import (
	"os"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// LogUTCFormatter hels in setting UTC time format for the logs
type LogUTCFormatter struct {
	log.Formatter
}

// Format helps fomratting time to UTC
func (u LogUTCFormatter) Format(e *log.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

func InitLog(cliLogLevel string) *log.Logger {

	// log instance for gin server
	log := logrus.New()
	log.SetFormatter(LogUTCFormatter{&logrus.JSONFormatter{}})

	if cliLogLevel == "" {
		cliLogLevel = os.Getenv("LOG_LEVEL")
	}

	var logLevel logrus.Level
	switch cliLogLevel {
	case "debug":
		logLevel = logrus.DebugLevel
	case "info":
		logLevel = logrus.InfoLevel
	case "warn":
		logLevel = logrus.WarnLevel
	case "error":
		logLevel = logrus.ErrorLevel
	case "fatal":
		logLevel = logrus.FatalLevel
	case "panic":
		logLevel = logrus.PanicLevel
	default:
		logLevel = logrus.InfoLevel
	}
	// set log level globally
	logrus.SetLevel(logLevel)

	// set log level for go-gin middleware
	log.SetLevel(logLevel)

	// show file path in log for debug or other log levels.
	if logLevel != logrus.InfoLevel {
		logrus.SetReportCaller(true)
		log.SetReportCaller(true)
	}

	return log
}
