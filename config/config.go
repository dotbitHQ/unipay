package config

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/fsnotify/fsnotify"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
	"sync"
	"time"
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
		Net      common.DasNetType `json:"net" yaml:"net"`
		HttpPort string            `json:"http_port" yaml:"http_port"`
		CronSpec string            `json:"cron_spec" yaml:"cron_spec"`
		//HedgeUrl         string            `json:"hedge_url" yaml:"hedge_url"`
		//RemoteSignApiUrl string            `json:"remote_sign_api_url" yaml:"remote_sign_api_url"`
	} `json:"server" yaml:"server"`
	BusinessIds map[string]string `json:"business_ids" yaml:"business_ids"`
	Notify      struct {
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

func GetPaymentAddress(payTokenId tables.PayTokenId) (string, error) {
	addr := ""
	switch payTokenId {
	case tables.PayTokenIdETH:
		addr = Cfg.Chain.Eth.Address
	case tables.PayTokenIdTRX:
		addr = Cfg.Chain.Tron.Address
	case tables.PayTokenIdBNB:
		addr = Cfg.Chain.Bsc.Address
	case tables.PayTokenIdMATIC:
		addr = Cfg.Chain.Polygon.Address
	case tables.PayTokenIdDAS, tables.PayTokenIdCKB:
		addr = Cfg.Chain.Ckb.Address
	default:
		return "", fmt.Errorf("unknow pay token id[%s]", payTokenId)
	}
	if addr == "" {
		return "", fmt.Errorf("payment address not configured")
	}
	return addr, nil
}

func InitDasCore(ctx context.Context, wg *sync.WaitGroup) (*core.DasCore, *dascache.DasCache, error) {
	// ckb node
	ckbClient, err := rpc.DialWithIndexer(Cfg.Chain.Ckb.Node, Cfg.Chain.Ckb.Node)
	if err != nil {
		return nil, nil, fmt.Errorf("rpc.DialWithIndexer err: %s", err.Error())
	}
	log.Info("ckb node ok")

	// das init
	net := Cfg.Server.Net
	env := core.InitEnvOpt(net, common.DasContractNameConfigCellType)
	ops := []core.DasCoreOption{
		core.WithClient(ckbClient),
		core.WithDasContractArgs(env.ContractArgs),
		core.WithDasContractCodeHash(env.ContractCodeHash),
		core.WithDasNetType(net),
		core.WithTHQCodeHash(env.THQCodeHash),
	}
	dasCore := core.NewDasCore(ctx, wg, ops...)
	dasCore.InitDasContract(env.MapContract)
	if err := dasCore.InitDasConfigCell(); err != nil {
		return nil, nil, fmt.Errorf("InitDasConfigCell err: %s", err.Error())
	}
	if err := dasCore.InitDasSoScript(); err != nil {
		return nil, nil, fmt.Errorf("InitDasSoScript err: %s", err.Error())
	}
	dasCore.RunAsyncDasContract(time.Minute * 3)   // contract outpoint
	dasCore.RunAsyncDasConfigCell(time.Minute * 5) // config cell outpoint
	dasCore.RunAsyncDasSoScript(time.Minute * 7)   // so

	log.Info("das contract ok")

	// das cache
	dasCache := dascache.NewDasCache(ctx, wg)
	dasCache.RunClearExpiredOutPoint(time.Minute * 15)
	log.Info("das cache ok")

	return dasCore, dasCache, nil
}
