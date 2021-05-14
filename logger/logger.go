package logger

import (
	"encoding/json"

	logger "github.com/astaxie/beego/logs"
)

type LoggerConfig struct {
	FileName            string `json:"filename"`
	Level               int    `json:"level"`    // the level when the log is saved, default: Trace
	Maxlines            int    `json:"maxlines"` // max line of files, default: 1000000
	Maxsize             int    `json:"maxsize"`  // max size of files, default: 1 << 28, //256 MB
	Daily               bool   `json:"daily"`    // whether logrotate daily，default: true
	Maxdays             int    `json:"maxdays"`  // max day file saved, default: 7 day
	Rotate              bool   `json:"rotate"`   // logrotate is enable, default: true
	Perm                string `json:"perm"`     // log file authority
	RotatePerm          string `json:"rotateperm"`
	EnableFuncCallDepth bool   `json:"-"` // output file name and line number
	LogFuncCallDepth    int    `json:"-"` // function call level
}

var logCfg = LoggerConfig{
	FileName:            "/var/log/terminal/terminal.log",
	Level:               7,
	EnableFuncCallDepth: true,
	LogFuncCallDepth:    3,
	RotatePerm:          "777",
	Perm:                "777",
	Maxdays:             90,
	Rotate:              true,
}

func init() {
	// set beego log package config
	logger.NewLogger(10000) // create a logger, param is cache size
	b, _ := json.Marshal(&logCfg)
	logger.SetLogger(logger.AdapterFile, string(b))
	logger.SetLogFuncCall(logCfg.EnableFuncCallDepth)
	logger.SetLogFuncCallDepth(logCfg.LogFuncCallDepth)
}
