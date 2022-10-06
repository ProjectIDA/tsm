package log

import (
	"fmt"
	"log/syslog"
	"os"
)

var logger *syslog.Writer
var logLevel syslog.Priority

// InitLogging subsystem. Caller must call Logger.Close() to close connection to syslog daemon
func InitLogging(tag string, defaultLogLevel syslog.Priority) error {

	if logger != nil {
		logger.Close()
	}

	log, err := syslog.New(defaultLogLevel, tag)
	if err != nil {
		logger = nil
		return err
	}
	logLevel = defaultLogLevel & 0b0111 // log level is first 3 bits of syslog flag set
	logger = log
	DebugMsg(fmt.Sprintf("priority: %bg; log level: %bg", defaultLogLevel, logLevel))
	return nil
}

// SetLogLevel sets a new default logging level
func SetLogLevel(newlvl syslog.Priority) {
	logLevel = newlvl & 0b0111
}

// Msg for loggin msg if requested level is <= current logLevel
func logMsg(lvl syslog.Priority, msg string) {
	if lvl <= logLevel {
		switch lvl {
		case syslog.LOG_EMERG:
			logger.Emerg(msg)
		case syslog.LOG_ALERT:
			logger.Alert(msg)
		case syslog.LOG_CRIT:
			logger.Crit(msg)
		case syslog.LOG_ERR:
			logger.Err(msg)
		case syslog.LOG_WARNING:
			logger.Warning(msg)
		case syslog.LOG_NOTICE:
			logger.Notice(msg)
		case syslog.LOG_INFO:
			logger.Info(msg)
		case syslog.LOG_DEBUG:
			logger.Debug(msg)
		default:
			logger.Notice(msg)
		}
	}
}

// EmergMsg log Emerg msessages
func EmergMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_EMERG, fmt.Sprintf(msgfmt, a...))
}

// AlertMsg log Emerg msessages
func AlertMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_ALERT, fmt.Sprintf(msgfmt, a...))
}

// CritMsg log Emerg msessages
func CritMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_CRIT, fmt.Sprintf(msgfmt, a...))
}

// ErrMsg log Emerg msessages
func ErrMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_ERR, fmt.Sprintf(msgfmt, a...))
}

// WarningMsg log Emerg msessages
func WarningMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_WARNING, fmt.Sprintf(msgfmt, a...))
}

// NoticeMsg log Emerg msessages
func NoticeMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_NOTICE, fmt.Sprintf(msgfmt, a...))
}

// InfoMsg log Emerg msessages
func InfoMsg(msgfmt string, a ...interface{}) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_INFO, fmt.Sprintf(msgfmt, a...))
}

// DebugMsg log Emerg msessages
func DebugMsg(msgfmt string, a ...interface{}) {
	// func DebugMsg(msg string) {
	if logger == nil {
		fmt.Fprintln(os.Stderr, "logger is nil. You must call InitLogging()")
		return
	}
	logMsg(syslog.LOG_DEBUG, "DEBUG: "+fmt.Sprintf(msgfmt, a...))
}
