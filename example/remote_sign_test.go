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
	"github.com/dotbitHQ/das-lib/remote_sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/shopspring/decimal"
	"testing"
	"unipay/config"
)

var (
	remoteSignUrl = "http://127.0.0.1:9093/v1/remote/sign"
)

func TestRemoteSignEVM(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), nodePolygon, addFee)
	if err != nil {
		t.Fatal(err)
	}
	from := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	to := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	value := decimal.NewFromInt(2 * 1e15)
	data := []byte("")

	nonce, err := chainEvm.NonceAt(from)
	if err != nil {
		t.Fatal(err)
	}
	gasPrice, gasLimit, err := chainEvm.EstimateGas(from, to, value, data, addFee)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := chainEvm.NewTransaction(from, to, value, data, nonce, gasPrice, gasLimit)
	if err != nil {
		t.Fatal(err)
	}

	chainID, err := chainEvm.Client.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	sigTx, err := remote_sign.SignTxForEVM(chainID.Int64(), remoteSignUrl, from, tx)
	if err != nil {
		t.Fatal(err)
	}

	if err = chainEvm.SendTransaction(sigTx); err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", sigTx.Hash().String())
}

func TestRemoteSignTron(t *testing.T) {
	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	from := "TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1"
	fromHex, _ := common.TronBase58ToHex(from)
	toHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	memo := ""
	amount := int64(10 * 1e6)
	tx, err := chainTron.CreateTransaction(fromHex, toHex, memo, amount)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := chain_tron.GetTxHash(tx)
	if err != nil {
		t.Fatal(err)
	}

	signData, err := remote_sign.SignTxForTRON(remoteSignUrl, from, hash)
	if err != nil {
		t.Fatal(err)
	}

	tx.Transaction.Signature = append(tx.Transaction.Signature, signData)
	tx.Txid = hash

	err = chainTron.SendTransaction(tx.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(tx.Txid))
}

func TestRemoteSignDoge(t *testing.T) {
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
	orderId := ""
	addrFrom := "DP86MSmWjEZw8GKotxcvAaW5D4e3qoEh6f"
	addrTo := "DQaRQ9s28U7EogPcDZudwZc4wD1NucZr2g"
	payAmount := int64(400000000)

	_, uos, err := txTool.GetUnspentOutputsDoge(addrFrom, "", payAmount)
	if err != nil {
		t.Fatal(err)
	}

	// transfer
	tx, err := txTool.NewTx(uos, []string{addrTo}, []int64{payAmount}, orderId)
	if err != nil {
		t.Fatal(err)
	}

	sigTx, err := remote_sign.SignTxForDOGE(remoteSignUrl, addrFrom, tx)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := txTool.SendTx(sigTx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", hash)
}

func TestRemoteSignCkb(t *testing.T) {
	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}

	amount := uint64(100) * common.OneCkb
	fee := uint64(1e6)
	orderid := ""

	fromAddr := "ckt1qyqvsej8jggu4hmr45g4h8d9pfkpd0fayfksz44t9q"
	fromParseAddress, err := address.Parse(fromAddr)
	if err != nil {
		t.Fatal(err)
	}
	toAddr := "ckt1qyqrekdjpy72kvhp3e9uf6y5868w5hjg8qnsqt6a0m"

	handleSign := remote_sign.SignTxForCKB(remoteSignUrl, fromAddr)
	txBuilderBase := txbuilder.NewDasTxBuilderBase(context.Background(), dc, handleSign, common.Bytes2Hex(fromParseAddress.Script.Args))

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
