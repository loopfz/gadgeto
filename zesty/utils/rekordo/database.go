package rekordo

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/loopfz/gadgeto/zesty"
)

// Default database settings.
const (
	maxOpenConns = 5
	maxIdleConns = 3
)

// DatabaseConfig represents the configuration used to
// register a new database.
type DatabaseConfig struct {
	Name             string
	DSN              string
	System           DBMS
	MaxOpenConns     int
	MaxIdleConns     int
	AutoCreateTables bool
}

// RegisterDatabase creates a gorp map with tables and tc and
// registers it with zesty.
func RegisterDatabase(db *DatabaseConfig, tc gorp.TypeConverter) error {
	dbConn, err := sql.Open(db.System.DriverName(), db.DSN)
	if err != nil {
		return err
	}
	// Make sure we have proper values for the database
	// settings, and replace them with default if necessary
	// before applying to the new connection.
	if db.MaxOpenConns == 0 {
		db.MaxOpenConns = maxOpenConns
	}
	dbConn.SetMaxOpenConns(db.MaxOpenConns)
	if db.MaxIdleConns == 0 {
		db.MaxIdleConns = maxIdleConns
	}
	dbConn.SetMaxIdleConns(db.MaxIdleConns)

	// Select the proper dialect used by gorp.
	var dialect gorp.Dialect
	switch db.System {
	case DatabaseMySQL:
		dialect = gorp.MySQLDialect{}
	case DatabasePostgreSQL:
		dialect = gorp.PostgresDialect{}
	case DatabaseSqlite3:
		dialect = gorp.SqliteDialect{}
	default:
		return errors.New("unknown database system")
	}
	dbmap := &gorp.DbMap{
		Db:            dbConn,
		Dialect:       dialect,
		TypeConverter: tc,
	}
	modelsMu.Lock()
	tableModels := models[db.Name]
	modelsMu.Unlock()
	for _, t := range tableModels {
		dbmap.AddTableWithName(t.Model, t.Name).SetKeys(t.AutoIncrement, t.Keys...)
	}

	if db.AutoCreateTables {
		err = dbmap.CreateTablesIfNotExists()
		if err != nil {
			return err
		}
	}
	return zesty.RegisterDB(zesty.NewDB(dbmap), db.Name)
}

// DBMS represents a database management system.
type DBMS uint8

// Database management systems.
const (
	DatabasePostgreSQL DBMS = iota ^ 42
	DatabaseMySQL
	DatabaseSqlite3
)

// DriverName returns the name of the driver for ds.
func (d DBMS) DriverName() string {
	switch d {
	case DatabasePostgreSQL:
		return "postgres"
	case DatabaseMySQL:
		return "mysql"
	case DatabaseSqlite3:
		return "sqlite3"
	}
	return ""
}
