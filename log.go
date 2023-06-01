// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package mogusocket

import (
	"fmt"
	"os"
	"strings"
)

// Logger is a simple logger interface that can have subloggers for specific areas.
type Logger interface {
	Warnf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Fatalf(msg string, args ...interface{})
	Warn(msgs ...interface{})
	Error(msgs ...interface{})
	Info(msgs ...interface{})
	Debug(msgs ...interface{})

	Sub(module string) Logger
}

type noopLogger struct{}

func (n *noopLogger) Errorf(_ string, _ ...interface{}) {}
func (n *noopLogger) Warnf(_ string, _ ...interface{})  {}
func (n *noopLogger) Infof(_ string, _ ...interface{})  {}
func (n *noopLogger) Debugf(_ string, _ ...interface{}) {}
func (n *noopLogger) Fatalf(_ string, _ ...interface{}) { os.Exit(1) }
func (n *noopLogger) Warn(_ ...interface{})             {}
func (n *noopLogger) Error(_ ...interface{})            {}
func (n *noopLogger) Info(_ ...interface{})             {}
func (n *noopLogger) Debug(_ ...interface{})            {}

func (n *noopLogger) Sub(_ string) Logger { return n }

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
	fmt.Print(colorStart, "[", s.mod, " ", level, "]")
	fmt.Println(outmsg, colorReset)
}

func (s *stdoutLogger) output(level string, msgs ...interface{}) {
	if levelToInt[level] < s.min {
		return
	}
	if !s.color {
		fmt.Print("[", s.mod, " ", level, "]")
		fmt.Println(msgs...)
	} else {
		fmt.Print(colors[level], "[", s.mod, " ", level, "]")
		msgs = append(msgs, "\033[0m")
		fmt.Println(msgs...)
	}
}

func (s *stdoutLogger) Fatalf(msg string, args ...interface{}) {
	s.outputf("ERROR", msg, args...)
	os.Exit(1)
}
func (s *stdoutLogger) Errorf(msg string, args ...interface{}) { s.outputf("ERROR", msg, args...) }
func (s *stdoutLogger) Warnf(msg string, args ...interface{})  { s.outputf("WARN", msg, args...) }
func (s *stdoutLogger) Infof(msg string, args ...interface{})  { s.outputf("INFO", msg, args...) }
func (s *stdoutLogger) Debugf(msg string, args ...interface{}) { s.outputf("DEBUG", msg, args...) }
func (s *stdoutLogger) Warn(msgs ...interface{})               { s.output("WARN", msgs...) }
func (s *stdoutLogger) Error(msgs ...interface{})              { s.output("ERROR", msgs...) }
func (s *stdoutLogger) Info(msgs ...interface{})               { s.output("INFO", msgs...) }
func (s *stdoutLogger) Debug(msgs ...interface{})              { s.output("DEBUG", msgs...) }

func (s *stdoutLogger) Sub(mod string) Logger {
	return &stdoutLogger{mod: fmt.Sprintf("%s/%s", s.mod, mod), color: s.color, min: s.min}
}

// Stdout is a simple Logger implementation that outputs to stdout. The module name given is included in log lines.
//
// minLevel specifies the minimum log level to output. An empty string will output all logs.
//
// If color is true, then info, warn and error logs will be colored cyan, yellow and red respectively using ANSI color escape codes.
func Stdout(module string, minLevel string, color bool) Logger {
	return &stdoutLogger{mod: module, color: color, min: levelToInt[strings.ToUpper(minLevel)]}
}

func LogSubName(_ string) {}
