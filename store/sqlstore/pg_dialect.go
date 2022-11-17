package sqlstore

import (
	"database/sql"
	"net/http"
	"reflect"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/webitel/storage/model"
)

const ForeignKeyViolationErrorCode = pq.ErrorCode("23503")
const DuplicationViolationErrorCode = pq.ErrorCode("23505")

type PostgresJSONDialect struct {
	gorp.PostgresDialect
}

func (d PostgresJSONDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	if val == reflect.TypeOf((*model.JSON)(nil)) {
		return "JSONB"
	}
	return d.PostgresDialect.ToSqlType(val, maxsize, isAutoIncr)
}

func extractCodeFromErr(err error) int {
	code := http.StatusInternalServerError

	if err == sql.ErrNoRows {
		code = http.StatusNotFound
	} else if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ForeignKeyViolationErrorCode, DuplicationViolationErrorCode:
			code = http.StatusBadRequest
		}
	}
	return code
}
