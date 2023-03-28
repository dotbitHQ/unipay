package dao

import (
	"github.com/dotbitHQ/unipay/tables"
	"gorm.io/gorm/clause"
)

func (d *DbDao) FindBlockInfoByBlockNumber(parserType tables.ParserType, blockNumber uint64) (block tables.TableBlockParserInfo, err error) {
	err = d.db.Where("parser_type=? AND block_number=?", parserType, blockNumber).Find(&block).Error
	return
}

func (d *DbDao) FindBlockInfo(parserType tables.ParserType) (block tables.TableBlockParserInfo, err error) {
	err = d.db.Where("parser_type=?", parserType).
		Order("block_number DESC").Limit(1).Find(&block).Error
	return
}

func (d *DbDao) CreateBlockInfoList(list []tables.TableBlockParserInfo) error {
	return d.db.Clauses(clause.Insert{
		Modifier: "IGNORE",
	}).Create(&list).Error
}

func (d *DbDao) DeleteBlockInfo(parserType tables.ParserType, blockNumber uint64) error {
	return d.db.Where("parser_type=? AND block_number < ?", parserType, blockNumber).
		Delete(&tables.TableBlockParserInfo{}).Error
}

func (d *DbDao) DeleteBlockInfoByBlockNumber(parserType tables.ParserType, blockNumber uint64) error {
	return d.db.Where("parser_type=? AND block_number=?", parserType, blockNumber).
		Delete(&tables.TableBlockParserInfo{}).Error
}

func (d *DbDao) CreateBlockInfo(blockInfo tables.TableBlockParserInfo) error {
	return d.db.Clauses(clause.Insert{
		Modifier: "IGNORE",
	}).Create(&blockInfo).Error
}
