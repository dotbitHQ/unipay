package refund

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/shopspring/decimal"
	"strings"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolRefund) doRefundCkb(paymentAddress, private string, list []tables.ViewRefundPaymentInfo) error {
	if !config.Cfg.Chain.Ckb.Refund {
		return fmt.Errorf("ckb refund flag is false")
	}
	if len(list) == 0 {
		return nil
	}
	log.Info("doRefundCkb:", paymentAddress, len(list))
	fromScript := common.GetNormalLockScript(paymentAddress)
	//
	dasContract, err := core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	balanceContract, err := core.GetDasContractInfo(common.DasContractNameBalanceCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	var txParams txbuilder.BuildTransactionParams
	totalAmount := decimal.Zero
	var payHashList []string
	for _, v := range list {
		if v.Amount.Cmp(decimal.Zero) != 1 {
			continue
		}
		ckbAddr, err := address.Parse(v.PayAddress)
		if err != nil {
			return fmt.Errorf("address.Parse err: %s", err.Error())
		}
		// NOTE check amount
		payHashList = append(payHashList, v.PayHash)
		totalAmount = totalAmount.Add(v.Amount)
		output := types.CellOutput{
			Capacity: v.Amount.BigInt().Uint64(),
			Lock:     ckbAddr.Script,
			Type:     nil,
		}
		if dasContract.IsSameTypeId(ckbAddr.Script.CodeHash) {
			output.Type = balanceContract.ToScript(nil)
		}
		txParams.Outputs = append(txParams.Outputs, &output)
		txParams.OutputsData = append(txParams.OutputsData, []byte(""))
	}

	// inputs
	inputsAmount := totalAmount.BigInt().Uint64()
	liveCells, total, err := t.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          nil,
		LockScript:        fromScript,
		CapacityNeed:      inputsAmount + common.OneCkb,
		CapacityForChange: common.MinCellOccupiedCkb,
		SearchOrder:       indexer.SearchOrderDesc,
	})
	if err != nil {
		return fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}
	for _, v := range liveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			Since:          0,
			PreviousOutput: v.OutPoint,
		})
	}

	// change
	if change := total - inputsAmount; change > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: change,
			Lock:     fromScript,
			Type:     nil,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte(""))
	}

	// witness
	actionWitness, err := witness.GenActionDataWitness("order_refund", nil)
	if err != nil {
		return fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// tx
	//txBuilderBase, err := config.InitDasTxBuilderBase(t.Ctx, t.DasCore, fromScript, private)
	txBuilderBase, err := config.InitDasTxBuilderBaseV2(t.Ctx, t.DasCore, fromScript, private)
	if err != nil {
		return fmt.Errorf("InitDasTxBuilderBase err: %s", err.Error())
	}
	txBuilder := txbuilder.NewDasTxBuilderFromBase(txBuilderBase, nil)
	if err := txBuilder.BuildTransaction(&txParams); err != nil {
		return fmt.Errorf("BuildTransaction err: %s", err.Error())
	}

	// fee
	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	log.Info("doRefundCKB sizeInBlock:", sizeInBlock)
	feeIndex := len(txBuilder.Transaction.Outputs) - 1
	changeCapacity := txBuilder.Transaction.Outputs[feeIndex].Capacity
	changeCapacity = changeCapacity - sizeInBlock - 5000
	txBuilder.Transaction.Outputs[feeIndex].Capacity = changeCapacity

	// send tx
	refundHash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return fmt.Errorf("ComputeHash err: %s", err.Error())
	}
	if err := t.DbDao.UpdatePaymentListToRefunded(payHashList, refundHash.Hex()); err != nil {
		return fmt.Errorf("UpdatePaymentListToRefunded err: %s", err.Error())
	}
	if _, err = txBuilder.SendTransaction(); err != nil {
		if err1 := t.DbDao.UpdatePaymentListToUnRefunded(payHashList); err1 != nil {
			log.Info("UpdatePaymentListToUnRefunded err: ", err1.Error(), payHashList)
			notify.SendLarkErrNotify("doRefundCKB", fmt.Sprintf("%s\n%s", strings.Join(payHashList, ","), err1.Error()))
		}
		return fmt.Errorf("SendTransaction err: %s", err.Error())
	}

	// callback notice
	if err = t.addCallbackNotice(list); err != nil {
		log.Error("addCallbackNotice err:", err.Error())
	}
	return nil
}
