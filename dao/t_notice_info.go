package dao

import (
	"github.com/dotbitHQ/unipay/tables"
	"time"
)

func (d *DbDao) Get24HUnNotifyList() (list []tables.TableNoticeInfo, err error) {
	nowTimestamp := time.Now().Add(-time.Hour * 24).Unix()
	err = d.db.Where("timestamp>=? AND notice_status=?",
		nowTimestamp, tables.NoticeStatusDefault).Find(&list).Error
	return
}

func (d *DbDao) UpdateNoticeStatus(id uint64, oldStatus, newStatus tables.NoticeStatus) error {
	return d.db.Model(tables.TableNoticeInfo{}).
		Where("id=? AND notice_status=?", id, oldStatus).
		Updates(map[string]interface{}{
			"notice_status": newStatus,
		}).Error
}

func (d *DbDao) UpdateNoticeCount(notice tables.TableNoticeInfo) error {
	notice.NoticeCount++
	if notice.NoticeCount > 3 {
		return d.db.Model(tables.TableNoticeInfo{}).
			Where("id=? AND notice_status=?", notice.Id, tables.NoticeStatusDefault).
			Updates(map[string]interface{}{
				"notice_status": tables.NoticeStatusFail,
			}).Error
	}
	return d.db.Model(tables.TableNoticeInfo{}).
		Where("id=? AND notice_status=?", notice.Id, tables.NoticeStatusDefault).
		Updates(map[string]interface{}{
			"notice_count": notice.NoticeCount,
		}).Error
}
