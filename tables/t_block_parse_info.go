package tables

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"time"
)

type TableBlockParserInfo struct {
	Id          uint64     `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	ParserType  ParserType `json:"parser_type" gorm:"column:parser_type; uniqueIndex:uk_parser_number; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '';"`
	BlockNumber uint64     `json:"block_number" gorm:"column:block_number; uniqueIndex:uk_parser_number; type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '';"`
	BlockHash   string     `json:"block_hash" gorm:"column:block_hash; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ParentHash  string     `json:"parent_hash" gorm:"column:parent_hash; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameBlockParserInfo = "t_block_parser_info"
)

func (t *TableBlockParserInfo) TableName() string {
	return TableNameBlockParserInfo
}

type ParserType int

const (
	ParserTypeCKB     ParserType = 0
	ParserTypeETH     ParserType = 1
	ParserTypeTRON    ParserType = 3
	ParserTypeBSC     ParserType = 5
	ParserTypePOLYGON ParserType = 6
	ParserTypeDoge    ParserType = 7
	ParserTypeDP      ParserType = 8
	ParserTypeBTC     ParserType = 9
	//ParserTypeDAS     = 99
)

func (p ParserType) ToString() string {
	switch p {
	case ParserTypeCKB:
		return "CKB"
	case ParserTypeETH:
		return "ETH"
	case ParserTypeTRON:
		return "TRON"
	case ParserTypeBSC:
		return "BSC"
	case ParserTypePOLYGON:
		return "POLYGON"
	case ParserTypeDoge:
		return "DOGE"
	case ParserTypeDP:
		return "DP"
	case ParserTypeBTC:
		return "BTC"
	}
	return fmt.Sprintf("%d", p)
}

func (p ParserType) ToAlgorithmId() common.DasAlgorithmId {
	switch p {
	case ParserTypeCKB, ParserTypeDP:
		return common.DasAlgorithmIdCkb
	case ParserTypeETH, ParserTypeBSC, ParserTypePOLYGON:
		return common.DasAlgorithmIdEth712
	case ParserTypeTRON:
		return common.DasAlgorithmIdTron
	case ParserTypeDoge:
		return common.DasAlgorithmIdDogeChain
	case ParserTypeBTC:
		return common.DasAlgorithmIdBitcoin
	}
	return -1
}
