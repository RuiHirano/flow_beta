package util

import (
	"log"
)

////////////////////////////////////////////////////////////
/////////////        Logger Class               ////////////
///////////////////////////////////////////////////////////

type Logger struct {
	Prefix string
}

func NewLogger() *Logger {
	//log.SetFlags(0)
	return &Logger{Prefix: ""}
}

func (l *Logger) SetPrefix(prefix string) {
	l.Prefix = prefix
}

func (l *Logger) Info(format string, args ...interface{}) {
	//log.SetPrefix(l.Prefix)
	//log.SetFlags(0)
	log.Printf("\x1b[32m\x1b[40m [Info] \x1b[0m"+format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	//log.SetPrefix(l.Prefix)
	//log.SetFlags(0)
	log.Printf("\x1b[31m\x1b[40m [Error] \x1b[0m"+format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	//log.SetPrefix(l.Prefix)
	//log.SetFlags(0)
	log.Printf("\x1b[33m\x1b[40m [Warn] \x1b[0m"+format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	//log.SetPrefix(l.Prefix)
	//log.SetFlags(0)
	log.Fatalf("\x1b[31m\x1b[40m [Error] \x1b[0m"+format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	//log.SetPrefix(l.Prefix)
	//log.SetFlags(0)
	log.Printf("\x1b[36m\x1b[40m [Debug] \x1b[0m"+format, args...)
}
