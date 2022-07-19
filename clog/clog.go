/*
Copyright 2021 KubeCube Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clog

import (
	"flag"
	"github.com/kubecube-io/kubecube/pkg/clog"
)

var (
	logFile         = *flag.String("log-file", "/etc/logs/cube.log", "")
	maxSize         = *flag.Int("max-size", 1000, "")
	maxBackups      = *flag.Int("max-backups", 7, "")
	maxAge          = *flag.Int("max-age", 1, "")
	compress        = *flag.Bool("compress", true, "")
	logLevel        = *flag.String("log-level", "info", "")
	jsonEncode      = *flag.Bool("json-encode", false, "")
	stacktraceLevel = *flag.String("stacktrace-level", "error", "")
)

func NewLogConfig() *clog.Config {
	flag.Parse()
	return &clog.Config{
		LogFile:         logFile,
		MaxSize:         maxSize,
		MaxBackups:      maxBackups,
		MaxAge:          maxAge,
		Compress:        compress,
		LogLevel:        logLevel,
		JsonEncode:      jsonEncode,
		StacktraceLevel: stacktraceLevel,
	}
}
