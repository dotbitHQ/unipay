package dao

import (
	"gorm.io/gorm/clause"
	"time"
	"unipay/tables"
)

func (d *DbDao) Get24HUnNotifyList() (list []tables.TableNoticeInfo, err error) {
	nowTimestamp := time.Now().Add(-time.Hour * 24).Unix()
	err = d.db.Where("timestamp>=? AND notice_status=?",
		nowTimestamp, tables.NoticeStatusDefault).Find(&list).Error
	return
}

func (d *DbDao) UpdateNoticeStatusToOK(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return d.db.Model(tables.TableNoticeInfo{}).
		Where("id IN(?) AND notice_status=?", ids, tables.NoticeStatusDefault).
		Updates(map[string]interface{}{
			"notice_status": tables.NoticeStatusOK,
		}).Error
}

func (d *DbDao) UpdateNoticeCount(id uint64, noticeCount int) error {
	return d.db.Model(tables.TableNoticeInfo{}).
		Where("id=? AND notice_status=?", id, tables.NoticeStatusDefault).
		Updates(map[string]interface{}{
			"notice_count": noticeCount,
		}).Error
}

func (d *DbDao) UpdateNoticeStatusToFail(id uint64) error {
	return d.db.Model(tables.TableNoticeInfo{}).
		Where("id=? AND notice_status=?", id, tables.NoticeStatusDefault).
		Updates(map[string]interface{}{
			"notice_status": tables.NoticeStatusFail,
		}).Error
}

func (d *DbDao) CreateNoticeList(list []tables.TableNoticeInfo) error {
	return d.db.Clauses(clause.Insert{
		Modifier: "IGNORE",
	}).Create(&list).Error
}
