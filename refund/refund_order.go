package refund

import "fmt"

func (t *ToolRefund) doRefund() error {
	// get refund list
	_, err := t.DbDao.GetRefundListWithin3d()
	if err != nil {
		return fmt.Errorf("GetRefundListWithin3d err: %s", err.Error())
	}

	return nil
}
