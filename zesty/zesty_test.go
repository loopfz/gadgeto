package zesty

import (
	"database/sql"
	"testing"

	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbName = "test"
	value1 = 1
	value2 = 2
)

func TestTransaction(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	RegisterDB(
		NewDB(&gorp.DbMap{
			Db:      db,
			Dialect: gorp.SqliteDialect{},
		}),
		dbName,
	)
	dbp, err := NewDBProvider(dbName)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dbp.DB().Exec(`CREATE TABLE "t" (id BIGINT);`)
	if err != nil {
		t.Fatal(err)
	}

	// first transaction: insert value 1
	sp0, err := dbp.TxSavepoint()
	if err != nil {
		t.Fatal(err)
	}
	if sp0 != 0 {
		t.Fatal("first transaction savepoint should be == 0")
	}

	_, err = dbp.DB().Exec(`INSERT INTO "t" VALUES (?)`, value1)
	if err != nil {
		t.Fatal(err)
	}

	// second transaction: update value to 2
	sp1, err := dbp.TxSavepoint()
	if err != nil {
		t.Fatal(err)
	}
	if sp1 != 1 {
		t.Fatal("first transaction savepoint should be == 1")
	}

	_, err = dbp.DB().Exec(`UPDATE "t" SET id = ?`, value2)
	if err != nil {
		t.Fatal(err)
	}

	i, err := dbp.DB().SelectInt(`SELECT id FROM "t"`)
	if err != nil {
		t.Fatal(err)
	}
	if i != value2 {
		t.Fatal("unexpected value found in table, expecting 2")
	}

	// rollback on second transaction: value back to 1
	err = dbp.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	i, err = dbp.DB().SelectInt(`SELECT id FROM "t"`)
	if err != nil {
		t.Fatal(err)
	}
	if i != value1 {
		t.Fatal("unexpected value found in table, expecting 1")
	}

	// noop rollback: savepoint already removed in previous rollback
	err = dbp.RollbackTo(sp1)
	if err != nil {
		t.Fatal("rollback to previous savepoint should return nil")
	}

	// rollback on first transaction: empty table
	err = dbp.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	j, err := dbp.DB().SelectNullInt(`SELECT id FROM "t"`)
	if err != nil {
		t.Fatal(err)
	}
	if j.Valid {
		t.Fatal("wrong value, was expecting empty sql.NullInt64 (no rows found)")
	}

	// no rollback possible after exiting outermost Tx
	err = dbp.Rollback()
	if err == nil {
		t.Fatal("rollback should fail when there is no transaction")
	}
}
