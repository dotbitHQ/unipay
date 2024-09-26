package config

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/fsnotify/fsnotify"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/stripe/stripe-go/v74"
	"strings"
	"sync"
	"time"
	"unipay/tables"
)

var (
	Cfg CfgServer
	log = logger.NewLogger("config", logger.LevelDebug)
)

func InitCfg(configFilePath string) error {
	if configFilePath == "" {
		configFilePath = "./config/config.yaml"
	}
	log.Debug("config file path：", configFilePath)
	if err := toolib.UnmarshalYamlFile(configFilePath, &Cfg); err != nil {
		return fmt.Errorf("UnmarshalYamlFile err:%s", err.Error())
	}
	log.Debug("config file：", toolib.JsonString(Cfg))
	initStripe()
	return nil
}

func AddCfgFileWatcher(configFilePath string) (*fsnotify.Watcher, error) {
	if configFilePath == "" {
		configFilePath = "./config/config.yaml"
	}
	return toolib.AddFileWatcher(configFilePath, func() {
		log.Debug("config file path：", configFilePath)
		if err := toolib.UnmarshalYamlFile(configFilePath, &Cfg); err != nil {
			log.Error("UnmarshalYamlFile err:", err.Error())
		}
		log.Debug("config file：", toolib.JsonString(Cfg))
		initStripe()
	})
}

type CfgServer struct {
	Server struct {
		Name                  string            `json:"name" yaml:"name"`
		Net                   common.DasNetType `json:"net" yaml:"net"`
		HttpPort              string            `json:"http_port" yaml:"http_port"`
		CronSpec              string            `json:"cron_spec" yaml:"cron_spec"`
		RemoteSignApiUrl      string            `json:"remote_sign_api_url" yaml:"remote_sign_api_url"`
		PrometheusPushGateway string            `json:"prometheus_push_gateway" yaml:"prometheus_push_gateway"`
	} `json:"server" yaml:"server"`
	BusinessIds map[string]string `json:"business_ids" yaml:"business_ids"`
	Notify      struct {
		LarkErrorKey   string `json:"lark_error_key" yaml:"lark_error_key"`
		LarkDasInfoKey string `json:"lark_das_info_key" yaml:"lark_das_info_key"`
		StripeKey      string `json:"stripe_key" yaml:"stripe_key"`
	} `json:"notify" yaml:"notify"`
	DB struct {
		Mysql DbMysql `json:"mysql" yaml:"mysql"`
	} `json:"db" yaml:"db"`
	Chain struct {
		DP struct {
			Refund                   bool   `json:"refund" yaml:"refund"`
			Switch                   bool   `json:"switch" yaml:"switch"`
			Node                     string `json:"node" yaml:"node"`
			CurrentBlockNumber       uint64 `json:"current_block_number" yaml:"current_block_number"`
			TransferWhitelist        string `json:"transfer_whitelist" yaml:"transfer_whitelist"`
			TransferWhitelistPrivate string `json:"transfer_whitelist_private" yaml:"transfer_whitelist_private"`
			RefundUrl                string `json:"refund_url" yaml:"refund_url"'`
		} `json:"dp" yaml:"dp"`
		Ckb struct {
			Refund          bool              `json:"refund" yaml:"refund"`
			Switch          bool              `json:"switch" yaml:"switch"`
			Node            string            `json:"node" yaml:"node"`
			AddrMap         map[string]string `json:"addr_map" yaml:"addr_map"`
			BalanceCheckMap map[string]string `json:"balance_check_map" yaml:"balance_check_map"`
		} `json:"ckb" yaml:"ckb"`
		Eth     EvmNode `json:"eth" yaml:"eth"`
		Tron    EvmNode `json:"tron" yaml:"tron"`
		Bsc     EvmNode `json:"bsc" yaml:"bsc"`
		Polygon EvmNode `json:"polygon" yaml:"polygon"`
		Doge    struct {
			TxChanNum int               `json:"tx_chan_num" yaml:"tx_chan_num"`
			Refund    bool              `json:"refund" yaml:"refund"`
			Switch    bool              `json:"switch" yaml:"switch"`
			Node      string            `json:"node" yaml:"node"`
			User      string            `json:"user" yaml:"user"`
			Password  string            `json:"password" yaml:"password"`
			Proxy     string            `json:"proxy" yaml:"proxy"`
			AddrMap   map[string]string `json:"addr_map" yaml:"addr_map"`
		} `json:"doge" yaml:"doge"`
		Stripe struct {
			Refund         bool   `json:"refund" yaml:"refund"`
			Switch         bool   `json:"switch" yaml:"switch"`
			Key            string `json:"key" yaml:"key"`
			EndpointSecret string `json:"endpoint_secret" yaml:"endpoint_secret"`
			WebhooksAddr   string `json:"webhooks_addr" yaml:"webhooks_addr"`
			LargeAmount    int64  `json:"large_amount" yaml:"large_amount"`
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
			if tronAddr, err := common.TronBase58ToHex(k); err != nil {
				log.Error("FormatAddrMap err:", parserType, k, err.Error())
				continue
			} else {
				res[tronAddr] = v
			}
		}
	case tables.ParserTypeCKB, tables.ParserTypeDP:
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

func GetPaymentAddress(payTokenId tables.PayTokenId, paymentAddress string) (string, error) {
	switch payTokenId {
	case tables.PayTokenIdETH, tables.PayTokenIdErc20USDT:
		if _, ok := Cfg.Chain.Eth.AddrMap[paymentAddress]; ok {
			return strings.ToLower(paymentAddress), nil
		}
	case tables.PayTokenIdTRX, tables.PayTokenIdTrc20USDT:
		if _, ok := Cfg.Chain.Tron.AddrMap[paymentAddress]; ok {
			if tronAddr, err := common.TronBase58ToHex(paymentAddress); err != nil {
				return "", fmt.Errorf("common.TronBase58ToHex err: %s[%s]", err.Error(), paymentAddress)
			} else {
				return tronAddr, nil
			}
		}
	case tables.PayTokenIdBNB, tables.PayTokenIdBep20USDT:
		if _, ok := Cfg.Chain.Bsc.AddrMap[paymentAddress]; ok {
			return strings.ToLower(paymentAddress), nil
		}
	case tables.PayTokenIdPOL: //,tables.PayTokenIdMATIC:
		if _, ok := Cfg.Chain.Polygon.AddrMap[paymentAddress]; ok {
			return strings.ToLower(paymentAddress), nil
		}
	case tables.PayTokenIdDAS, tables.PayTokenIdCKB, tables.PayTokenIdCkbCCC:
		if _, ok := Cfg.Chain.Ckb.AddrMap[paymentAddress]; ok {
			if parseAddr, err := address.Parse(paymentAddress); err != nil {
				return "", fmt.Errorf("address.Parse err: %s[%s]", err.Error(), paymentAddress)
			} else if parseAddr.Script.CodeHash.String() != transaction.SECP256K1_BLAKE160_SIGHASH_ALL_TYPE_HASH {
				return "", fmt.Errorf("Script.CodeHash Invaild: %s", paymentAddress)
			} else {
				return common.Bytes2Hex(parseAddr.Script.Args), nil
			}
		}
	case tables.PayTokenIdDOGE:
		if _, ok := Cfg.Chain.Doge.AddrMap[paymentAddress]; ok {
			return paymentAddress, nil
		}
	case tables.PayTokenIdStripeUSD:
		return "", nil
	case tables.PayTokenIdDIDPoint:
		return "", nil
	}
	return "", fmt.Errorf("unknow pay token id[%s] in AddrMap[%s]", payTokenId, paymentAddress)
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
	env := core.InitEnvOpt(net,
		common.DasContractNameConfigCellType,
		common.DasContractNameDispatchCellType,
		common.DasContractNameBalanceCellType,
		common.DASContractNameEip712LibCellType,
		common.DasContractNameDpCellType,
	)
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

func initStripe() {
	stripe.Key = Cfg.Chain.Stripe.Key
}

func InitDasTxBuilderBaseV2(ctx context.Context, dasCore *core.DasCore, fromScript *types.Script, private string) (*txbuilder.DasTxBuilderBase, error) {
	if fromScript == nil {
		return nil, fmt.Errorf("fromScript is nil")
	}
	svrArgs := common.Bytes2Hex(fromScript.Args)
	var handleSign sign.HandleSignCkbMessage
	if private != "" {
		handleSign = sign.LocalSign(private)
	} else if Cfg.Server.RemoteSignApiUrl != "" {
		mode := address.Testnet
		if Cfg.Server.Net == common.DasNetTypeMainNet {
			mode = address.Mainnet
		}
		addr, err := address.ConvertScriptToShortAddress(mode, fromScript)
		if err != nil {
			return nil, fmt.Errorf("address.ConvertScriptToShortAddress err: %s", err.Error())
		}
		handleSign = remote_sign.SignTxForCKBHandle(Cfg.Server.RemoteSignApiUrl, addr)
	}
	txBuilderBase := txbuilder.NewDasTxBuilderBase(ctx, dasCore, handleSign, svrArgs)
	return txBuilderBase, nil
}
