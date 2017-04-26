// zesty is based on gorp, and abstracts DB transaction specifics.
// You can create a zesty.DB by calling NewDB().
// You can then register this DB by calling RegisterDB().
// This lets you instantiate DBProviders for this DB with NewDBProvider(), which is the main
// object that you manipulate.
// A DBProvider contains a DB instance, and provides Tx functionalities.
// You access the DB by calling provider.DB()
// By calling provider.Tx(), you create a new transaction.
// Future calls to provider.DB() will provide the Tx instead of the main DB object,
// allowing caller code to be completely ignorant of transaction context.
// Transactions can be nested infinitely, and each nesting level can be rolled back independantly.
// Only the final commit will end the transaction and commit the changes to the DB.
package zesty

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/go-gorp/gorp"
)

// Registered databases
var (
	dbs    = make(map[string]DB)
	dblock sync.RWMutex
)

/*
 * INTERFACES
 */

type DB interface {
	gorp.SqlExecutor
	Begin() (Tx, error)
	Close() error
	Ping() error
	Stats() sql.DBStats
}

type Tx interface {
	gorp.SqlExecutor
	Commit() error
	Rollback() error
	Savepoint(string) error
	RollbackToSavepoint(string) error
}

type DBProvider interface {
	DB() gorp.SqlExecutor
	Tx() error
	Commit() error
	Rollback() error
	Close() error
	Ping() error
	Stats() sql.DBStats
}

/*
 * FUNCTIONS
 */

func NewDB(dbmap *gorp.DbMap) DB {
	return &zestydb{DbMap: dbmap}
}

func RegisterDB(db DB, name string) error {
	dblock.Lock()
	defer dblock.Unlock()

	_, ok := dbs[name]
	if ok {
		return fmt.Errorf("DB name conflict '%s'", name)
	}

	dbs[name] = db

	return nil
}

func UnregisterDB(name string) error {
	dblock.Lock()
	defer dblock.Unlock()

	_, ok := dbs[name]
	if !ok {
		return fmt.Errorf("No such database '%s'", name)
	}

	delete(dbs, name)

	return nil
}

func NewDBProvider(name string) (DBProvider, error) {
	dblock.RLock()
	defer dblock.RUnlock()
	db, ok := dbs[name]
	if !ok {
		return nil, fmt.Errorf("No such database '%s'", name)
	}
	return &zestyprovider{
		current: db,
		db:      db,
	}, nil
}

func NewTempDBProvider(db DB) DBProvider {
	return &zestyprovider{
		current: db,
		db:      db,
	}
}

/*
 * PROVIDER IMPLEMENTATION
 */

type zestyprovider struct {
	current gorp.SqlExecutor
	db      DB
	tx      Tx
	nested  int
}

func (zp *zestyprovider) DB() gorp.SqlExecutor {
	return zp.current
}

func (zp *zestyprovider) Commit() error {
	if zp.tx == nil {
		return errors.New("No active Tx")
	}

	if zp.nested > 0 {
		zp.nested--
		return nil
	}

	err := zp.tx.Commit()
	if err != nil {
		return err
	}

	zp.resetTx()

	return nil
}

func (zp *zestyprovider) Tx() error {
	if zp.tx != nil {
		s := fmt.Sprintf("tx-nested-%d", zp.nested+1)
		err := zp.tx.Savepoint(s)
		if err != nil {
			return err
		}
		zp.nested++
		return nil
	}

	tx, err := zp.db.Begin()
	if err != nil {
		return err
	}

	zp.tx = tx
	zp.current = tx

	return nil
}

func (zp *zestyprovider) Rollback() error {
	if zp.tx == nil {
		return errors.New("No active Tx")
	}

	if zp.nested > 0 {
		s := fmt.Sprintf("tx-nested-%d", zp.nested)
		err := zp.tx.RollbackToSavepoint(s)
		if err != nil {
			return err
		}
		zp.nested--
		return nil
	}

	err := zp.tx.Rollback()
	if err != nil {
		return err
	}

	zp.resetTx()

	return nil
}

func (zp *zestyprovider) resetTx() {
	zp.current = zp.db
	zp.tx = nil
}

func (zp *zestyprovider) Close() error {
	return zp.db.Close()
}

func (zp *zestyprovider) Ping() error {
	return zp.db.Ping()
}

func (zp *zestyprovider) Stats() sql.DBStats {
	return zp.db.Stats()
}

/*
 * DATABASE IMPLEMENTATION
 */

type zestydb struct {
	*gorp.DbMap
}

func (zd *zestydb) Begin() (Tx, error) {
	return zd.DbMap.Begin()
}

func (zd *zestydb) Close() error {
	return zd.DbMap.Db.Close()
}

func (zd *zestydb) Ping() error {
	return zd.DbMap.Db.Ping()
}

func (zd *zestydb) Stats() sql.DBStats {
	return zd.DbMap.Db.Stats()
}
