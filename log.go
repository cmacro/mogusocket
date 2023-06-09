// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package mogusocket

import (
	"fmt"
	"log"
	"strings"
)

// Logger is a simple logger interface that can have subloggers for specific areas.
type Logger interface {
	Warn(msgs ...interface{})
	Error(msgs ...interface{})
	Info(msgs ...interface{})
	Debug(msgs ...interface{})
}

type noopLogger struct{}

func (n *noopLogger) Warn(_ ...interface{})  {}
func (n *noopLogger) Error(_ ...interface{}) {}
func (n *noopLogger) Info(_ ...interface{})  {}
func (n *noopLogger) Debug(_ ...interface{}) {}

// Noop is a no-op Logger implementation that silently drops everything.
var Noop Logger = &noopLogger{}

type stdoutLogger struct {
	mod   string
	color bool
	min   int
}

var colors = map[string]string{
	"INFO":  "\033[36m",
	"WARN":  "\033[33m",
	"ERROR": "\033[31m",
}

var levelToInt = map[string]int{
	"":      -1,
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
}

var LevelToSeverity = map[string]int{
	"":      0,
	"DEBUG": 0,
	"INFO":  0,
	"WARN":  1,
	"ERROR": 2,
}

func (s *stdoutLogger) outputf(level, msg string, args ...interface{}) {
	if levelToInt[level] < s.min {
		return
	}
	var colorStart, colorReset string
	if s.color {
		colorStart = colors[level]
		colorReset = "\033[0m"
	}
	outmsg := msg
	if len(args) > 0 {
		outmsg = fmt.Sprintf(msg, args...)
	}
	log.Print(colorStart, "[", s.mod, " ", level, "] ", outmsg, colorReset)
	// log.Println(outmsg, colorReset)
	// log.Print(v ...any)
}

func (s *stdoutLogger) output(level string, msgs ...interface{}) {
	if levelToInt[level] < s.min {
		return
	}
	var colorStart, colorReset string
	if s.color {
		colorStart = colors[level]
		colorReset = "\033[0m"
	}
	outmsg := fmt.Sprintln(msgs...)
	log.Print(colorStart, "[", s.mod, " ", level, "] ", outmsg[:len(outmsg)-1], colorReset)
}

func (s *stdoutLogger) Warn(msgs ...interface{})  { s.output("WARN", msgs...) }
func (s *stdoutLogger) Error(msgs ...interface{}) { s.output("ERROR", msgs...) }
func (s *stdoutLogger) Info(msgs ...interface{})  { s.output("INFO", msgs...) }
func (s *stdoutLogger) Debug(msgs ...interface{}) { s.output("DEBUG", msgs...) }

// Stdout is a simple Logger implementation that outputs to stdout. The module name given is included in log lines.
//
// minLevel specifies the minimum log level to output. An empty string will output all logs.
//
// If color is true, then info, warn and error logs will be colored cyan, yellow and red respectively using ANSI color escape codes.
func Stdout(module string, minLevel string, color bool) Logger {
	return &stdoutLogger{mod: module, color: color, min: levelToInt[strings.ToUpper(minLevel)]}
}

func LogSubName(_ string) {}
