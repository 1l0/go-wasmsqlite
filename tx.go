//go:build js && wasm

package wasmsqlite

import (
	"database/sql/driver"
	"errors"
)

var ErrTxDone = errors.New("sql: transaction has already been committed or rolled back")

// Tx implements the database/sql/driver.Tx interface
type Tx struct {
	conn *Conn
}

// Commit implements driver.Tx
func (tx *Tx) Commit() error {
	if tx.conn.api == nil {
		return driver.ErrBadConn
	}

	if !tx.conn.inTx {
		return ErrTxDone
	}

	err := tx.conn.api.Commit()
	if err != nil {
		return err
	}

	tx.conn.inTx = false
	return nil
}

// Rollback implements driver.Tx
func (tx *Tx) Rollback() error {
	if tx.conn.api == nil {
		return driver.ErrBadConn
	}

	if !tx.conn.inTx {
		return ErrTxDone
	}

	err := tx.conn.api.Rollback()
	if err != nil {
		return err
	}

	tx.conn.inTx = false
	return nil
}
