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
	LogFile         = *flag.String("log-file", "/etc/logs/cube.log", "")
	MaxSize         = *flag.Int("max-size", 1000, "")
	MaxBackups      = *flag.Int("max-backups", 7, "")
	MaxAge          = *flag.Int("max-age", 1, "")
	Compress        = *flag.Bool("compress", true, "")
	LogLevel        = *flag.String("log-level", "info", "")
	JsonEncode      = *flag.Bool("json-encode", false, "")
	StacktraceLevel = *flag.String("stacktrace-level", "error", "")
)

func NewLogConfig() *clog.Config {
	return &clog.Config{
		LogFile:         LogFile,
		MaxSize:         MaxSize,
		MaxBackups:      MaxBackups,
		MaxAge:          MaxAge,
		Compress:        Compress,
		LogLevel:        LogLevel,
		JsonEncode:      JsonEncode,
		StacktraceLevel: StacktraceLevel,
	}
}
