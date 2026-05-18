package mysql

import "gorm.io/gorm"

type TxRunner interface {
	WithinTx(func(tx *gorm.DB) error) error
}
