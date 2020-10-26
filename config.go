package main

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

/**
 * local config struct
 */
type LocalConfig struct {
	Name             string    `json:"name"`
	LoggerLevel      *string   `json:"log_level"`
	VortexPort       *int      `json:"vortex_port"`
	DatabaseHost     *string   `json:"database_host"`
	DatabaseTable    *string   `json:"database_table"`
	DatabaseUsername *string   `json:"database_username"`
	DatabasePassword *string   `json:"database_password"`
	JwtSecret        *string   `json:"jwt_secret"`
	CacheDeadline    time.Time `json:"-"`
	Version          *string   `json:"version"`
	VersionTag       *string   `json:"versionTag"`
	IotPort          *int   `json:"iot_port"`
	IotVersion *IV `json:"iot_version"`
}

type IV struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"`
}

var config = new(LocalConfig)

func GetConfig() *LocalConfig {
	if config.CacheDeadline.Before(time.Now()) {
		if err := LoadConfig("conf.json", config); err != nil {
			Fatal("loading file failed")
			return nil
		}
		config.CacheDeadline = time.Now().Add(time.Second * 60)
	}
	return config
}

/**
 * save cnf/conf.json
 */
func (lc *LocalConfig) SetConfig() error {
	fp, err := os.Create("conf.json")
	if err != nil {
		Fatal("loading file failed", logrus.Fields{"err": err})
	}
	defer fp.Close()
	data, err := json.Marshal(lc)
	if err != nil {
		Fatal("marshal file failed", logrus.Fields{"err": err})
	}
	n, err := fp.Write(data)
	if err != nil {
		Fatal("write file failed", logrus.Fields{"err": err})
	}
	Info("already update config file", logrus.Fields{"size": n})
	return nil
}

func (lc *LocalConfig) GetDatabaseUsername() string {
	return *lc.DatabaseUsername
}

func (lc *LocalConfig) GetDatabasePassword() string {
	return *lc.DatabasePassword
}

func (lc *LocalConfig) GetDatabaseHost() string {
	return *lc.DatabaseHost
}

func (lc *LocalConfig) GetDatabaseTable() string {
	return *lc.DatabaseTable
}

func (lc *LocalConfig) GetJwtSecret() []byte {
	return []byte(*lc.JwtSecret)
}