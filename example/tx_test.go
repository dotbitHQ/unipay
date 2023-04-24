package example

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
	"testing"
	"unipay/config"
)

var (
	node, addFee = "https://rpc.ankr.com/eth_goerli", float64(1.5)
	nodeBsc      = "https://rpc.ankr.com/bsc_testnet_chapel"
	nodePolygon  = "https://rpc.ankr.com/polygon_mumbai"
	nodeTron     = "grpc.nile.trongrid.io:50051"
	privateKey   = ""
)

func TestTron(t *testing.T) {
	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	fromHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	toHex, _ := common.TronBase58ToHex("TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1")
	memo := "3d863f089368ccad5eb1e746417e2803"
	amount := int64(500 * 1e6)
	tx, err := chainTron.CreateTransaction(fromHex, toHex, memo, amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainTron.LocalSign(tx, privateKey); err != nil {
		t.Fatal(err)
	}

	err = chainTron.SendTransaction(tx.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(tx.Txid))
}

func TestCkbTx(t *testing.T) {
	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}

	amount := uint64(1024) * common.OneCkb
	fee := uint64(1e6)
	orderid := "a7dff2d50bdd053aee42e8f4fe3f17b1"

	fromAddr := "ckt1qyqvsej8jggu4hmr45g4h8d9pfkpd0fayfksz44t9q"
	fromParseAddress, err := address.Parse(fromAddr)
	if err != nil {
		t.Fatal(err)
	}
	toAddr := "ckt1qyqvsej8jggu4hmr45g4h8d9pfkpd0fayfksz44t9q"
	txBuilderBase := getTxBuilderBase(dc, common.Bytes2Hex(fromParseAddress.Script.Args), privateKey)
	toParseAddress, err := address.Parse(toAddr)
	if err != nil {
		t.Fatal(err)
	}
	liveCells, total, err := dc.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          nil,
		LockScript:        fromParseAddress.Script,
		CapacityNeed:      amount + fee,
		CapacityForChange: common.MinCellOccupiedCkb,
		SearchOrder:       indexer.SearchOrderAsc,
	})
	if err != nil {
		t.Fatal(err, total)
	}
	fmt.Println(len(liveCells))
	//
	var txParams txbuilder.BuildTransactionParams
	for i, v := range liveCells {
		fmt.Println(i)
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			Since:          0,
			PreviousOutput: v.OutPoint,
		})
	}
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: amount,
		Lock:     toParseAddress.Script,
		Type:     nil,
	})
	txParams.OutputsData = append(txParams.OutputsData, []byte(orderid))
	//

	if change := total - amount - fee; change > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: change,
			Lock:     fromParseAddress.Script,
			Type:     nil,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	//
	txBuilder := txbuilder.NewDasTxBuilderFromBase(txBuilderBase, nil)
	if err := txBuilder.BuildTransaction(&txParams); err != nil {
		t.Fatal(err)
	}

	if hash, err := txBuilder.SendTransactionWithCheck(false); err != nil {
		t.Fatal(err)
	} else {
		fmt.Println("hash:", hash)
	}
}

func TestEvmTx(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, addFee)
	if err != nil {
		t.Fatal(err)
	}
	from := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	to := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	value := decimal.NewFromInt(5407802377269926)
	data := []byte("569ffc5b4fc347194e49ac8d7a63f7c3")
	nonce, err := chainEvm.NonceAt(from)
	if err != nil {
		t.Fatal(err)
	}
	gasPrice, gasLimit, err := chainEvm.EstimateGas(from, to, value, data, addFee)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(gasPrice.Mul(gasLimit).String())
	tx, err := chainEvm.NewTransaction(from, to, value, data, nonce, gasPrice, gasLimit)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = chainEvm.SignWithPrivateKey(privateKey, tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = chainEvm.SendTransaction(tx); err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", tx.Hash().String())
}

func TestDogeTx(t *testing.T) {
	_ = config.InitCfg("../config/config.yaml")
	br := bitcoin.BaseRequest{
		RpcUrl:   config.Cfg.Chain.Doge.Node,
		User:     config.Cfg.Chain.Doge.User,
		Password: config.Cfg.Chain.Doge.Password,
		Proxy:    config.Cfg.Chain.Doge.Proxy,
	}
	txTool := bitcoin.TxTool{
		RpcClient:        &br,
		Ctx:              context.Background(),
		RemoteSignClient: nil,
		DustLimit:        bitcoin.DustLimitDoge,
		Params:           bitcoin.GetDogeMainNetParams(),
	}
	// get utxo
	addrFrom := ""
	addrTo := ""
	payAmount := int64(2e8)
	orderId := "778afd143d32dbfcef3a85accb8eda64"
	_, uos, err := txTool.GetUnspentOutputsDoge(addrFrom, privateKey, payAmount)
	if err != nil {
		t.Fatal(err)
	}

	// transfer
	tx, err := txTool.NewTx(uos, []string{addrTo}, []int64{payAmount}, orderId)
	if err != nil {
		t.Fatal(err)
	}

	_, err = txTool.LocalSignTx(tx, uos)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := txTool.SendTx(tx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", hash)
}

func TestErc20Tx(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, 5)
	if err != nil {
		t.Fatal(err)
	}
	from := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	to := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	contract := "0xBA62BCfcAaFc6622853cca2BE6Ac7d845BC0f2Dc"

	value := decimal.NewFromBigInt(new(big.Int).SetUint64(1e18), 0)
	fmt.Println(value.Coefficient().String())
	data, err := chain_evm.PackMessage("transfer", ethcommon.HexToAddress(to), value.Coefficient())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(common.Bytes2Hex(data))
	//0xa9059cbb00000000000000000000000015a33588908cf8edb27d1abe3852bf287abd38910000000000000000000000000000000000000000000000000de0b6b3a7640000
	//0xa9059cbb00000000000000000000000015a33588908cf8edb27d1abe3852bf287abd38910000000000000000000000000000000000000000000000000de0b6b3a7640000

	nonce, err := chainEvm.NonceAt(from)
	if err != nil {
		t.Fatal(err)
	}
	gasPrice, gasLimit, err := chainEvm.EstimateGas(from, contract, decimal.Zero, data, chainEvm.RefundAddFee)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(gasPrice.Mul(gasLimit).String())
	tx, err := chainEvm.NewTransaction(from, contract, decimal.Zero, data, nonce, gasPrice, gasLimit)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = chainEvm.SignWithPrivateKey(privateKey, tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = chainEvm.SendTransaction(tx); err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", tx.Hash().String())
}

func TestTx(t *testing.T) {
	node = "https://rpc.ankr.com/eth"
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, addFee)
	if err != nil {
		t.Fatal(err)
	}
	block, err := chainEvm.GetBlockByNumber(15790501)
	if err != nil {
		t.Fatal(err)
	}
	addr := ""
	for _, tx := range block.Transactions {
		if strings.EqualFold(tx.To, "0xdAC17F958D2ee523a2206206994597C13D831ec7") {
			fmt.Println(tx.From, tx.Input, tx.Value)
			fmt.Println(len(tx.Input), len(common.Hex2Bytes(tx.Input)), len(tx.Input[10:74]), len(addr))
			fmt.Println(tx.Input[:10])
			fmt.Println(tx.Input[10:74])
			fmt.Println(tx.Input[34:74])
			fmt.Println(tx.Input[74:])
			amount := decimal.NewFromBigInt(new(big.Int).SetBytes(common.Hex2Bytes(tx.Input)[36:]), 0)
			fmt.Println(amount.String())
		}
	}
}
