package sqlstore

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/webitel/storage/store"
)

type driver struct {
	db *sqlx.DB
}

func NewSqlDriver(db *sql.DB, driverName string) store.Driver {
	return &driver{
		db: sqlx.NewDb(db, driverName),
	}
}

func (d *driver) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

func (d *driver) SelectOne(holder interface{}, query string, args ...interface{}) error {
	rows, err := d.db.Queryx(query, args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	return rows.StructScan(&holder)
}

func (d *driver) Select(dest interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return []interface{}{}, d.db.Select(dest, query, args...)
}

func (d *driver) SelectInt(query string, args ...interface{}) (int64, error) {
	var h int64
	_, err := d.Select(&h, query, args)
	if err != nil {
		return 0, err
	}
	return h, nil
}

func (d *driver) SelectNullInt(query string, args ...interface{}) (sql.NullInt64, error) {
	var h sql.NullInt64
	_, err := d.Select(&h, query, args)
	if err != nil && err != sql.ErrNoRows {
		return h, err
	}
	return h, nil
}

func (d *driver) TraceOn() {

}
