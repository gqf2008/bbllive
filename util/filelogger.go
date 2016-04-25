package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type FileLogger struct {
	prefix string
	logger *log.Logger
	level  int
}

func (l *FileLogger) Debugf(format string, v ...interface{}) {
	if l.level > 0 {
		return
	}
	l.logger.Printf("DEBUG "+format, v...)
}

func (l *FileLogger) Debug(v ...interface{}) {
	if l.level > 0 {
		return
	}
	l.logger.Println("DEBUG", fmt.Sprintln(v...))
}

func (l *FileLogger) Infof(format string, v ...interface{}) {
	if l.level > 1 {
		return
	}
	l.logger.Printf("INFO "+format, v...)
}

func (l *FileLogger) Info(v ...interface{}) {
	if l.level > 1 {
		return
	}
	l.logger.Println("INFO", fmt.Sprintln(v...))
}

func (l *FileLogger) Warnf(format string, v ...interface{}) {
	if l.level > 2 {
		return
	}

	l.logger.Printf("WARN "+format, v...)
}

func (l *FileLogger) Warn(v ...interface{}) {
	if l.level > 2 {
		return
	}
	l.logger.Println("WARN", fmt.Sprintln(v...))
}

func (l *FileLogger) Errorf(format string, v ...interface{}) {
	if l.level > 3 {
		return
	}
	l.logger.Printf("ERROR "+format, v...)
}

func (l *FileLogger) Error(v ...interface{}) {
	if l.level > 3 {
		return
	}
	l.logger.Println("ERROR", fmt.Sprintln(v...))
}

func NewFileLogger(prefix, logfile string, level int) (*FileLogger, error) {
	if err := os.MkdirAll(filepath.Dir(logfile), os.ModePerm); err != nil {
		return nil, err
	}
	var logger *log.Logger
	if "" == logfile || "stdout" == logfile {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	} else {
		fi, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		logger = log.New(fi, "", log.Ldate|log.Ltime)
	}

	return &FileLogger{prefix, logger, level}, nil
}
