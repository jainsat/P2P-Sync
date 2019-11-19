package lib

import (
	"log"
	"os"
	"sync"
)

type logger struct {
	filename string
	*log.Logger
	mu sync.Mutex
}

var syncLogger *logger
var once sync.Once

// Get the instance of logger
func GetLogger() *logger {
	once.Do(func() {
		syncLogger = createLogger("/var/log/sync.log")
	})
	return syncLogger
}

func (l *logger) Debug(format string, a ...interface{}) {
	l.mu.Lock()
	l.Printf(format, a...)
	l.mu.Unlock()
}
func createLogger(fname string) *logger {
	file, _ := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)

	return &logger{
		filename: fname,
		Logger:   log.New(file, "[P2PSync]", log.Lshortfile),
	}
}
