package repository

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

const mysqlErrDuplicateEntry = 1062

func isDuplicateKeyErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == mysqlErrDuplicateEntry
	}
	return false
}
