package config

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/fsnotify/fsnotify"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
	"github.com/stripe/stripe-go/v74"
	"strings"
	"sync"
	"time"
	"unipay/tables"
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
	stripe.Key = Cfg.Chain.Stripe.Key
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
		stripe.Key = Cfg.Chain.Stripe.Key
	})
}

type CfgServer struct {
	Server struct {
		Net              common.DasNetType `json:"net" yaml:"net"`
		HttpPort         string            `json:"http_port" yaml:"http_port"`
		CronSpec         string            `json:"cron_spec" yaml:"cron_spec"`
		RemoteSignApiUrl string            `json:"remote_sign_api_url" yaml:"remote_sign_api_url"`
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
			Refund  bool              `json:"refund" yaml:"refund"`
			Switch  bool              `json:"switch" yaml:"switch"`
			Node    string            `json:"node" yaml:"node"`
			Address string            `json:"address" yaml:"address"`
			Private string            `json:"private" yaml:"private"`
			AddrMap map[string]string `json:"addr_map" yaml:"addr_map"`
		} `json:"ckb" yaml:"ckb"`
		Eth     EvmNode `json:"eth" yaml:"eth"`
		Tron    EvmNode `json:"tron" yaml:"tron"`
		Bsc     EvmNode `json:"bsc" yaml:"bsc"`
		Polygon EvmNode `json:"polygon" yaml:"polygon"`
		Doge    struct {
			Refund   bool              `json:"refund" yaml:"refund"`
			Switch   bool              `json:"switch" yaml:"switch"`
			Node     string            `json:"node" yaml:"node"`
			Address  string            `json:"address" yaml:"address"`
			Private  string            `json:"private" yaml:"private"`
			User     string            `json:"user" yaml:"user"`
			Password string            `json:"password" yaml:"password"`
			Proxy    string            `json:"proxy" yaml:"proxy"`
			AddrMap  map[string]string `json:"addr_map" yaml:"addr_map"`
		} `json:"doge" yaml:"doge"`
		Stripe struct {
			Refund         bool   `json:"refund" yaml:"refund"`
			Key            string `json:"key" yaml:"key"`
			EndpointSecret string `json:"endpoint_secret" yaml:"endpoint_secret"`
			WebhooksAddr   string `json:"webhooks_addr" yaml:"webhooks_addr"`
		} `json:"stripe" yaml:"stripe"`
	} `json:"chain" yaml:"chain"`
}

type DbMysql struct {
	Addr     string `json:"addr" yaml:"addr"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	DbName   string `json:"db_name" yaml:"db_name"`
}

type EvmNode struct {
	Refund       bool              `json:"refund" yaml:"refund"`
	Switch       bool              `json:"switch" yaml:"switch"`
	Node         string            `json:"node" yaml:"node"`
	Address      string            `json:"address" yaml:"address"`
	Private      string            `json:"private" yaml:"private"`
	RefundAddFee float64           `json:"refund_add_fee" yaml:"refund_add_fee"`
	AddrMap      map[string]string `json:"addr_map" yaml:"addr_map"`
}

func FormatAddrMap(parserType tables.ParserType, addrMap map[string]string) map[string]string {
	var res = make(map[string]string)
	switch parserType {
	case tables.ParserTypeETH, tables.ParserTypeBSC, tables.ParserTypePOLYGON:
		for k, v := range addrMap {
			res[strings.ToLower(k)] = v
		}
	case tables.ParserTypeTRON:
		for k, v := range addrMap {
			if strings.HasPrefix(k, common.TronBase58PreFix) {
				if tronAddr, err := common.TronBase58ToHex(k); err != nil {
					log.Error("FormatAddrMap err:", parserType, k, err.Error())
					continue
				} else {
					res[tronAddr] = v
				}
			}
		}
	case tables.ParserTypeCKB:
		for k, v := range addrMap {
			parseAddrK, err := address.Parse(k)
			if err != nil {
				log.Error("FormatAddrMap err:", parserType, k, err.Error())
				continue
			}
			res[common.Bytes2Hex(parseAddrK.Script.Args)] = v
		}
	default:
		res = addrMap
	}
	return res
}

func GetReceiptAddr(payTokenId tables.PayTokenId, receiptAddr string) (string, error) {
	addr := ""
	switch payTokenId {
	case tables.PayTokenIdETH, tables.PayTokenIdErc20USDT:
		if _, ok := Cfg.Chain.Eth.AddrMap[receiptAddr]; ok {
			return receiptAddr, nil
		}
	case tables.PayTokenIdTRX, tables.PayTokenIdTrc20USDT:
		if _, ok := Cfg.Chain.Tron.AddrMap[receiptAddr]; ok {
			return receiptAddr, nil
		}
	case tables.PayTokenIdBNB, tables.PayTokenIdBep20USDT:
		if _, ok := Cfg.Chain.Bsc.AddrMap[receiptAddr]; ok {
			return receiptAddr, nil
		}
	case tables.PayTokenIdMATIC:
		if _, ok := Cfg.Chain.Polygon.AddrMap[receiptAddr]; ok {
			return receiptAddr, nil
		}
	case tables.PayTokenIdDAS, tables.PayTokenIdCKB:
		if _, ok := Cfg.Chain.Ckb.AddrMap[receiptAddr]; ok {
			return receiptAddr, nil
		}
	case tables.PayTokenIdDOGE:
		if _, ok := Cfg.Chain.Doge.AddrMap[receiptAddr]; ok {
			return receiptAddr, nil
		}
	case tables.PayTokenIdStripeUSD:
		addr = "stripe"
	default:
		return "", fmt.Errorf("unknow pay token id[%s]", payTokenId)
	}
	if addr == "" {
		return "", fmt.Errorf("payment address not configured")
	}
	return addr, nil
}

func InitDasCore(ctx context.Context, wg *sync.WaitGroup) (*core.DasCore, *dascache.DasCache, *txbuilder.DasTxBuilderBase, error) {
	// ckb node
	ckbClient, err := rpc.DialWithIndexer(Cfg.Chain.Ckb.Node, Cfg.Chain.Ckb.Node)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("rpc.DialWithIndexer err: %s", err.Error())
	}
	log.Info("ckb node ok")

	// das init
	net := Cfg.Server.Net
	env := core.InitEnvOpt(net,
		common.DasContractNameConfigCellType,
		common.DasContractNameDispatchCellType,
		common.DasContractNameBalanceCellType)
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
		return nil, nil, nil, fmt.Errorf("InitDasConfigCell err: %s", err.Error())
	}
	if err := dasCore.InitDasSoScript(); err != nil {
		return nil, nil, nil, fmt.Errorf("InitDasSoScript err: %s", err.Error())
	}
	dasCore.RunAsyncDasContract(time.Minute * 3)   // contract outpoint
	dasCore.RunAsyncDasConfigCell(time.Minute * 5) // config cell outpoint
	dasCore.RunAsyncDasSoScript(time.Minute * 7)   // so

	log.Info("das contract ok")

	// das cache
	dasCache := dascache.NewDasCache(ctx, wg)
	dasCache.RunClearExpiredOutPoint(time.Minute * 15)
	log.Info("das cache ok")

	//
	payServerAddressArgs := ""
	if Cfg.Chain.Ckb.Address != "" {
		parseAddress, err := address.Parse(Cfg.Chain.Ckb.Address)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("address.Parse err: %s", err.Error())
		} else {
			payServerAddressArgs = common.Bytes2Hex(parseAddress.Script.Args)
		}
	}
	var handleSign sign.HandleSignCkbMessage
	if Cfg.Chain.Ckb.Private != "" {
		handleSign = sign.LocalSign(Cfg.Chain.Ckb.Private)
	} else if Cfg.Server.RemoteSignApiUrl != "" && payServerAddressArgs != "" {
		remoteSignClient, err := sign.NewClient(ctx, Cfg.Server.RemoteSignApiUrl)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("sign.NewClient err: %s", err.Error())
		}
		handleSign = sign.RemoteSign(remoteSignClient, Cfg.Server.Net, payServerAddressArgs)
	}
	txBuilderBase := txbuilder.NewDasTxBuilderBase(ctx, dasCore, handleSign, payServerAddressArgs)

	return dasCore, dasCache, txBuilderBase, nil
}
