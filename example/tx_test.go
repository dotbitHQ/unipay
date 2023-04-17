package example

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/shopspring/decimal"
	"testing"
)

var (
	node, addFee = "https://rpc.ankr.com/eth_goerli", float64(1.5)
	privateKey   = ""
)

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
