package example

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	ethcommon "github.com/ethereum/go-ethereum/common"
	trxCore "github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/robfig/cron/v3"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
	"testing"
	"time"
	"unipay/config"
)

var (
	node, addFee = "https://rpc.ankr.com/eth_goerli", float64(2)
	nodeBsc      = "https://rpc.ankr.com/bsc_testnet_chapel"
	nodePolygon  = "https://rpc.ankr.com/polygon_mumbai"
	nodeTron     = "grpc.trongrid.io:50051"
	privateKey   = ""
	privateKey2  = ""
)

func TestTrc20(t *testing.T) {
	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	contractHex, _ := common.TronBase58ToHex("TKMVcZtc1kyb2qFruhgd91mRCPNhPRRrsw")
	fromHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	toHex, _ := common.TronBase58ToHex("TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1")
	tx, err := chainTron.TransferTrc20(contractHex, fromHex, toHex, 4*1e6, 20*1e6)
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
func TestTrc202(t *testing.T) {
	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	contractHex, _ := common.TronBase58ToHex("TKMVcZtc1kyb2qFruhgd91mRCPNhPRRrsw")
	fromHex, _ := common.TronBase58ToHex("TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1")
	toHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	tx, err := chainTron.TransferTrc20(contractHex, fromHex, toHex, 3*1e6, 20*1e6)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainTron.LocalSign(tx, privateKey2); err != nil {
		t.Fatal(err)
	}

	err = chainTron.SendTransaction(tx.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(tx.Txid))
}

func TestTrc203(t *testing.T) {
	TestTrc20(t)
	TestTrc202(t)
}

func TestTrc20Tx(t *testing.T) {
	contractHex, _ := common.TronBase58ToHex("TKMVcZtc1kyb2qFruhgd91mRCPNhPRRrsw")
	fromHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	toHex, _ := common.TronBase58ToHex("TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1")
	fmt.Println(contractHex)
	fmt.Println(fromHex)
	fmt.Println(toHex)

	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	block, err := chainTron.GetBlockByNumber(70510010)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range block.Transactions {
		if len(v.Transaction.RawData.Contract) != 1 {
			continue
		}
		switch v.Transaction.RawData.Contract[0].Type {
		case trxCore.Transaction_Contract_TriggerSmartContract:
			fmt.Println(hex.EncodeToString(v.Txid))
			smart := trxCore.TriggerSmartContract{}
			if err := proto.Unmarshal(v.Transaction.RawData.Contract[0].Parameter.Value, &smart); err != nil {
				t.Fatal(err)
			}
			fHex, cHex := hex.EncodeToString(smart.OwnerAddress), hex.EncodeToString(smart.ContractAddress)
			fmt.Println(fHex) // from
			fmt.Println(cHex) // contract
			fmt.Println(len(smart.Data))
			data := common.Bytes2Hex(smart.Data)
			fmt.Println(data)
			// a9059cbb is the hex str of transfer
			if len(smart.Data) != 68 || !strings.Contains(data, "a9059cbb") {
				continue
			}
			fmt.Println(hex.EncodeToString(smart.Data[0:16]))
			tHex := hex.EncodeToString(smart.Data[16:36])
			fmt.Println(tHex)
			amount := decimal.NewFromBigInt(new(big.Int).SetBytes(smart.Data[36:]), 0)
			fmt.Println(amount)
			//if !strings.EqualFold(data[34:74], addr[2:]) {
			//	continue
			//}
		}
	}
}

func TestTron(t *testing.T) {
	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	fromHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	toHex, _ := common.TronBase58ToHex("TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1")
	memo := "ae46417aa85b46f2acec5adf4e09af37"
	amount := int64(10 * 1e6)
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

func TestTron2(t *testing.T) {
	chainTron, err := chain_tron.NewChainTron(context.Background(), nodeTron)
	if err != nil {
		t.Fatal(err)
	}
	fromHex, _ := common.TronBase58ToHex("TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1")
	toHex, _ := common.TronBase58ToHex("TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV")
	memo := "ca36c94c29a7946a7e0772e178945ea7"
	amount := int64(9 * 1e6)
	tx, err := chainTron.CreateTransaction(fromHex, toHex, memo, amount)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainTron.LocalSign(tx, privateKey2); err != nil {
		t.Fatal(err)
	}

	err = chainTron.SendTransaction(tx.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(tx.Txid))
}

func TestTron3(t *testing.T) {
	TestTron(t)
	TestTron2(t)
}

func TestCkbTx(t *testing.T) {
	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}

	amount := uint64(100) * common.OneCkb
	fee := uint64(1e6)
	orderid := "0bc21198b2307b74bf2045304bd8981a"

	fromAddr := "ckt1qyqvsej8jggu4hmr45g4h8d9pfkpd0fayfksz44t9q"
	fromParseAddress, err := address.Parse(fromAddr)
	if err != nil {
		t.Fatal(err)
	}
	toAddr := "ckt1qyqrekdjpy72kvhp3e9uf6y5868w5hjg8qnsqt6a0m"
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

func TestCKBTx(t *testing.T) {
	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}

	orderId := "0bc21198b2307b74bf2045304bd8981a"
	fromAddr := "ckt1qyqvsej8jggu4hmr45g4h8d9pfkpd0fayfksz44t9q"
	privateFrom := privateKey
	toAddr := "ckt1qyqrekdjpy72kvhp3e9uf6y5868w5hjg8qnsqt6a0m"
	amount := 100 * common.OneCkb
	if err = txCkb(dc, orderId, fromAddr, privateFrom, toAddr, amount); err != nil {
		t.Fatal(err)
	}

	orderId = "14e713f8d2134f3b5d4855cc60f0dd22"
	fromAddr = "ckt1qyqrekdjpy72kvhp3e9uf6y5868w5hjg8qnsqt6a0m"
	privateFrom = privateKey2
	toAddr = "ckt1qyqvsej8jggu4hmr45g4h8d9pfkpd0fayfksz44t9q"
	amount = 110 * common.OneCkb
	if err = txCkb(dc, orderId, fromAddr, privateFrom, toAddr, amount); err != nil {
		t.Fatal(err)
	}
}

func txCkb(dc *core.DasCore, orderId, fromAddr, privateFrom, toAddr string, amount uint64) error {
	fee := uint64(1e6)
	fromPA, err := address.Parse(fromAddr)
	if err != nil {
		return fmt.Errorf("address.Parse err: %s", err.Error())
	}

	txBuilderBase := getTxBuilderBase(dc, common.Bytes2Hex(fromPA.Script.Args), privateFrom)
	toPA, err := address.Parse(toAddr)
	if err != nil {
		return fmt.Errorf("address.Parse err: %s", err.Error())
	}
	liveCells, total, err := dc.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          nil,
		LockScript:        fromPA.Script,
		CapacityNeed:      amount + fee,
		CapacityForChange: common.MinCellOccupiedCkb,
		SearchOrder:       indexer.SearchOrderAsc,
	})
	if err != nil {
		return fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}
	//fmt.Println(len(liveCells))
	//
	var txParams txbuilder.BuildTransactionParams
	for _, v := range liveCells {
		//fmt.Println(i)
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			Since:          0,
			PreviousOutput: v.OutPoint,
		})
	}
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: amount,
		Lock:     toPA.Script,
		Type:     nil,
	})
	txParams.OutputsData = append(txParams.OutputsData, []byte(orderId))
	//

	if change := total - amount - fee; change > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: change,
			Lock:     fromPA.Script,
			Type:     nil,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	//
	txBuilder := txbuilder.NewDasTxBuilderFromBase(txBuilderBase, nil)
	if err := txBuilder.BuildTransaction(&txParams); err != nil {
		return fmt.Errorf("BuildTransaction err: %s", err.Error())
	}

	if hash, err := txBuilder.SendTransactionWithCheck(false); err != nil {
		return fmt.Errorf("SendTransactionWithCheck err: %s", err.Error())
	} else {
		fmt.Println("hash:", hash)
	}
	return nil
}

func TestEvmTx3(t *testing.T) {
	go TestEvmTx(t)
	go TestEvmTx2(t)
	time.Sleep(10 * time.Second)
}

func TestEvmTx(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, addFee)
	if err != nil {
		t.Fatal(err)
	}
	from := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	to := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	value := decimal.NewFromInt(2 * 1e15)
	data := []byte("cba67295de7eaca7bd3d424e55127c96")

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

func TestEvmTx2(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, addFee)
	if err != nil {
		t.Fatal(err)
	}
	from := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	to := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	value := decimal.NewFromInt(3 * 1e15)
	data := []byte("377028345605517609b25b10c4112652")

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
	tx, err = chainEvm.SignWithPrivateKey(privateKey2, tx)
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

func TestDogeTx2(t *testing.T) {
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
	orderId := "fea8de748162bad438f40ba9c3af6d6e"
	addrFrom := "DQaRQ9s28U7EogPcDZudwZc4wD1NucZr2g"
	addrTo := "DP86MSmWjEZw8GKotxcvAaW5D4e3qoEh6f"
	fromKey := privateKey
	payAmount := int64(400000000)
	go func() {
		if err := txDoge(txTool, orderId, addrFrom, addrTo, fromKey, payAmount); err != nil {
			t.Fatal(err)
		}
	}()

	orderId2 := "698470be9a9dd2665de160f63f8a91f3"
	addrFrom2 := "DP86MSmWjEZw8GKotxcvAaW5D4e3qoEh6f"
	addrTo2 := "DQaRQ9s28U7EogPcDZudwZc4wD1NucZr2g"
	fromKey2 := privateKey2
	payAmount2 := int64(300000000)
	go func() {
		if err := txDoge(txTool, orderId2, addrFrom2, addrTo2, fromKey2, payAmount2); err != nil {
			t.Fatal(err)
		}
	}()
	time.Sleep(time.Second * 15)
}

func txDoge(txTool bitcoin.TxTool, orderId, addrFrom, addrTo, fromKey string, payAmount int64) error {
	_, uos, err := txTool.GetUnspentOutputsDoge(addrFrom, fromKey, payAmount)
	if err != nil {
		return fmt.Errorf("GetUnspentOutputsDoge err: %s", err.Error())
	}

	// transfer
	tx, err := txTool.NewTx(uos, []string{addrTo}, []int64{payAmount}, orderId)
	if err != nil {
		return fmt.Errorf("NewTx err: %s", err.Error())
	}

	_, err = txTool.LocalSignTx(tx, uos)
	if err != nil {
		return fmt.Errorf("LocalSignTx err: %s", err.Error())
	}

	hash, err := txTool.SendTx(tx)
	if err != nil {
		return fmt.Errorf("SendTx err: %s", err.Error())
	}
	fmt.Println("hash:", hash)
	return nil
}

func TestDogeTx3(t *testing.T) {
	_ = config.InitCfg("../config/config.yaml")
	br := bitcoin.BaseRequest{
		RpcUrl:   config.Cfg.Chain.Doge.Node,
		User:     config.Cfg.Chain.Doge.User,
		Password: config.Cfg.Chain.Doge.Password,
		Proxy:    config.Cfg.Chain.Doge.Proxy,
	}
	data, err := br.GetRawTransaction("ccd286a447d16cf5f166edac21b4b5f25df45612c19569043c1283d97fdc0189")
	if err != nil {
		t.Fatal(err)
	}
	var orderId string
	for _, vOut := range data.Vout {
		switch vOut.ScriptPubKey.Type {
		case txscript.NullDataTy.String():
			asm := vOut.ScriptPubKey.Asm
			orderId = strings.TrimPrefix(asm, "OP_RETURN ")
			if len(orderId) == 64 {
				bys, _ := hex.DecodeString(orderId)
				orderId = string(bys)
			}
		}
	}
	fmt.Println("orderId:", orderId)
	//
	//sc := txscript.NewScriptBuilder()
	//sc.AddOp(txscript.OP_RETURN).AddData([]byte("fea8de748162bad438f40ba9c3af6d6e"))
	//bs, err := sc.Script()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//res := wire.NewTxOut(0, bs)
	//tx := wire.NewMsgTx(wire.TxVersion)
	//tx.AddTxOut(res)
	//buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	//if err := tx.SerializeNoWitness(buf); err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Println(hex.EncodeToString(buf.Bytes()))
}

func TestErc20Tx(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, addFee)
	if err != nil {
		t.Fatal(err)
	}
	from := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"     //"0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	to := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"       //"0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	contract := "0x5Efb0D565898be6748920db2c3BdC22BDFd5c187" //"0xDf954C7D93E300183836CdaA01a07a1743F183EC"

	value := decimal.NewFromBigInt(new(big.Int).SetUint64(5*1e6), 0)
	fmt.Println(value.Coefficient().String())
	data, err := chain_evm.PackMessage("transfer", ethcommon.HexToAddress(to), value.Coefficient())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(common.Bytes2Hex(data))

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

func TestErc20Tx2(t *testing.T) {
	chainEvm, err := chain_evm.NewChainEvm(context.Background(), node, addFee)
	if err != nil {
		t.Fatal(err)
	}
	from := "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"     //"0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"
	to := "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891"       //"0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7"
	contract := "0x5Efb0D565898be6748920db2c3BdC22BDFd5c187" //"0xDf954C7D93E300183836CdaA01a07a1743F183EC"

	value := decimal.NewFromBigInt(new(big.Int).SetUint64(4*1e6), 0)
	fmt.Println(value.Coefficient().String())
	data, err := chain_evm.PackMessage("transfer", ethcommon.HexToAddress(to), value.Coefficient())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(common.Bytes2Hex(data))

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
	tx, err = chainEvm.SignWithPrivateKey(privateKey2, tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = chainEvm.SendTransaction(tx); err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash:", tx.Hash().String())
}

func TestErc20Tx3(t *testing.T) {
	go TestErc20Tx(t)
	go TestErc20Tx2(t)
	time.Sleep(10 * time.Second)
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

func TestCron(t *testing.T) {
	c := cron.New(cron.WithSeconds())
	_, err := c.AddFunc("0 30 */1 * * *", func() {
		fmt.Println(time.Now().String())
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Start()
	select {}
}
