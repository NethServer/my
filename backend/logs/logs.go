package logs

import (
	"log"
	"os"
)

var Logs *log.Logger

func Init(name string) {
	logger := log.New(os.Stderr, name+" ", log.Ldate|log.Ltime|log.Lshortfile)
	Logs = logger
}
