package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var Logger zerolog.Logger

func Init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	timeFormat := "2006-01-02 15:04:05"
	zerolog.TimeFieldFormat = timeFormat

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	logDir := exePath + string(os.PathSeparator) + "log" + string(os.PathSeparator)
	err = os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		fmt.Println("Mkdir failed, err:", err)
		return
	}
	fileName := logDir + time.Now().Format("2006-01-02") + ".log"
	logFile, _ := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	multi := zerolog.MultiLevelWriter(logFile)
	Logger = zerolog.New(multi).With().Timestamp().Logger().With().Caller().Logger()
}
