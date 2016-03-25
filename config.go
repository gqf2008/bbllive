package bbllive

import (
	log "github.com/cihub/seelog"
	"github.com/kless/goconfig/config"
	"os"
	"path/filepath"
)

var Config *config.Config
var main_config_file string

func config_init(file string) error {
	f, err := os.Open(file)
	defer f.Close()
	if err == nil {
		Config, err = config.ReadDefault(file)
		main_config_file = file
	}
	return err
}

func log_init() {
	defer log.Flush()
	logger, e := log.LoggerFromConfigAsFile(GetConfigDir() + "/log.conf")
	if e != nil {
		log.Criticalf("Error %v", e)
	}
	log.ReplaceLogger(logger)
	//TODO timer watch config file
}

func GetConfigDir() string {
	return filepath.Dir(main_config_file)
}

func GetString(section, key, def string) string {
	v, err := Config.String(section, key)
	if err != nil {
		return def
	}
	return v
}

func GetInt(section, key string, def int) int {
	v, err := Config.Int(section, key)
	if err != nil {
		return def
	}
	return v
}

func GetFloat(section, key string, def float64) float64 {
	v, err := Config.Float(section, key)
	if err != nil {
		return def
	}
	return v
}

func GetBool(section, key string, def bool) bool {
	v, err := Config.Bool(section, key)
	if err != nil {
		return def
	}
	return v
}
