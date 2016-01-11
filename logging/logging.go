/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
 *
 * Author(s):
 *    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * $BEGIN_LICENSE:AGPL3+$
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * $END_LICENSE$
 ***************************************************************************/

package logging

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	TraceLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

// Just to make the compiler happy: it fails because os is imported
// and used only by init() but it doesn't sees it
var out = os.Stdout

//func (l *loggingT) Init(traceHandle io.Writer, infoHandle io.Writer, warningHandle io.Writer, errorHandle io.Writer) {
func init() {
	// Create log file
	logFileName := os.Args[0] + ".log"
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		log.Fatalf("Failed to open log file \"%s\": %s", logFileName, err.Error())
	}

	// Write to the log file and output at the same time
	multi := io.MultiWriter(logFile, os.Stdout)

	// Setup loggers
	TraceLogger = log.New(multi, fmt.Sprintf("%sTRACE%s ", magenta, reset), log.Ldate|log.Ltime)
	InfoLogger = log.New(multi, fmt.Sprintf("%sINFO%s ", blue, reset), log.Ldate|log.Ltime)
	WarningLogger = log.New(multi, fmt.Sprintf("%sWARNING%s ", yellow, reset), log.Ldate|log.Ltime)
	ErrorLogger = log.New(multi, fmt.Sprintf("%sERROR%s ", red, reset), log.Ldate|log.Ltime)
}

func Trace(args ...interface{}) {
	TraceLogger.Print(args...)
}

func Traceln(args ...interface{}) {
	TraceLogger.Println(args...)
}

func Tracef(format string, args ...interface{}) {
	TraceLogger.Printf(format, args...)
}

func Info(args ...interface{}) {
	InfoLogger.Print(args...)
}

func Infoln(args ...interface{}) {
	InfoLogger.Println(args...)
}

func Infof(format string, args ...interface{}) {
	InfoLogger.Printf(format, args...)
}

func Warning(args ...interface{}) {
	WarningLogger.Print(args...)
}

func Warningln(args ...interface{}) {
	WarningLogger.Println(args...)
}

func Warningf(format string, args ...interface{}) {
	WarningLogger.Printf(format, args...)
}

func Error(args ...interface{}) {
	ErrorLogger.Print(args...)
}

func Errorln(args ...interface{}) {
	ErrorLogger.Println(args...)
}

func Errorf(format string, args ...interface{}) {
	ErrorLogger.Printf(format, args...)
}

func Fatal(args ...interface{}) {
	ErrorLogger.Fatal(args...)
}

func Fatalln(args ...interface{}) {
	ErrorLogger.Fatalln(args...)
}

func Fatalf(format string, args ...interface{}) {
	ErrorLogger.Fatalf(format, args...)
}
