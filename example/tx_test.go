package example

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/shopspring/decimal"
	"testing"
)

var (
	node, addFee = "https://rpc.ankr.com/eth_goerli", float64(1.5)
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
	amount := int64(1e6)
	tx, err := chainTron.CreateTransaction(fromHex, toHex, memo, amount)
	if err != nil {
		t.Fatal(err)
	}

	txSign, err := chainTron.AddSign(tx.Transaction, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	hash := hex.EncodeToString(txSign.Txid)
	fmt.Println("tx hash:", hash)
	err = chainTron.SendTransaction(txSign.Transaction)
	if err != nil {
		t.Fatal(err)
	}
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
	value := decimal.NewFromInt(1e16)
	data := []byte("f6d42e7e07bee6aab966c9c4e8f39045")
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
	tx, err = chainEvm.SignWithPrivateKey(privateKey, tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = chainEvm.SendTransaction(tx); err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", tx.Hash().String())
}
