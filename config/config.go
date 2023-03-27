package config

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/fsnotify/fsnotify"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
)

var (
	Cfg CfgServer
	log = mylog.NewLogger("config", mylog.LevelDebug)
)

func InitCfg(configFilePath string) error {
	if configFilePath == "" {
		configFilePath = "./config/config.yaml"
	}
	log.Info("config file path：", configFilePath)
	if err := toolib.UnmarshalYamlFile(configFilePath, &Cfg); err != nil {
		return fmt.Errorf("UnmarshalYamlFile err:%s", err.Error())
	}
	log.Info("config file：", toolib.JsonString(Cfg))
	return nil
}

func AddCfgFileWatcher(configFilePath string) (*fsnotify.Watcher, error) {
	if configFilePath == "" {
		configFilePath = "./config/config.yaml"
	}
	return toolib.AddFileWatcher(configFilePath, func() {
		log.Info("config file path：", configFilePath)
		if err := toolib.UnmarshalYamlFile(configFilePath, &Cfg); err != nil {
			log.Error("UnmarshalYamlFile err:", err.Error())
		}
		log.Info("config file：", toolib.JsonString(Cfg))
	})
}

type CfgServer struct {
	Server struct {
		Net common.DasNetType `json:"net" yaml:"net"`
		//CronSpec         string            `json:"cron_spec" yaml:"cron_spec"`
		//HedgeUrl         string            `json:"hedge_url" yaml:"hedge_url"`
		//RemoteSignApiUrl string            `json:"remote_sign_api_url" yaml:"remote_sign_api_url"`
	} `json:"server" yaml:"server"`
	Notify struct {
		LarkErrorKey string `json:"lark_error_key" yaml:"lark_error_key"`
	} `json:"notify" yaml:"notify"`
	DB struct {
		Mysql DbMysql `json:"mysql" yaml:"mysql"`
	} `json:"db" yaml:"db"`
	Chain struct {
		Ckb struct {
			Refund  bool   `json:"refund" yaml:"refund"`
			Switch  bool   `json:"switch" yaml:"switch"`
			Node    string `json:"node" yaml:"node"`
			Address string `json:"address" yaml:"address"`
			Private string `json:"private" yaml:"private"`
		} `json:"ckb"`
		Eth     EvmNode `json:"eth" yaml:"eth"`
		Tron    EvmNode `json:"tron" yaml:"tron"`
		Bsc     EvmNode `json:"bsc" yaml:"bsc"`
		Polygon EvmNode `json:"polygon" yaml:"polygon"`
		Doge    struct {
			Refund   bool   `json:"refund" yaml:"refund"`
			Switch   bool   `json:"switch" yaml:"switch"`
			Node     string `json:"node" yaml:"node"`
			Address  string `json:"address" yaml:"address"`
			Private  string `json:"private" yaml:"private"`
			User     string `json:"user" yaml:"user"`
			Password string `json:"password" yaml:"password"`
		} `json:"doge" yaml:"doge"`
	} `json:"chain" yaml:"chain"`
}

type DbMysql struct {
	Addr     string `json:"addr" yaml:"addr"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	DbName   string `json:"db_name" yaml:"db_name"`
}

type EvmNode struct {
	Refund       bool    `json:"refund" yaml:"refund"`
	Switch       bool    `json:"switch" yaml:"switch"`
	Node         string  `json:"node" yaml:"node"`
	Address      string  `json:"address" yaml:"address"`
	Private      string  `json:"private" yaml:"private"`
	RefundAddFee float64 `json:"refund_add_fee" yaml:"refund_add_fee"`
}
