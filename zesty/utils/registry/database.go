package utils

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/loopfz/gadgeto/zesty"
)

// DatabaseConfig represents the configuration used to
// register a new database.
type DatabaseConfig struct {
	Name         string
	DSN          string
	System       DBMS
	MaxOpenConns int
	MaxIdleConns int
}

// RegisterDatabase creates a gorp map with tables and tc and
// registers it with zesty.
func RegisterDatabase(db *DatabaseConfig, tc gorp.TypeConverter) error {
	tableModels, ok := models[db.Name]
	if !ok {
		return fmt.Errorf("no models registered for database %s", db.Name)
	}
	dbConn, err := sql.Open(db.System.DriverName(), db.DSN)
	if err != nil {
		return err
	}
	dbConn.SetMaxOpenConns(db.MaxOpenConns)
	dbConn.SetMaxIdleConns(db.MaxIdleConns)

	// Select the proper dialect used by gorp.
	var dialect gorp.Dialect
	switch db.System {
	case SystemMySQL:
		dialect = gorp.MySQLDialect{}
	case SystemPostgreSQL:
		dialect = gorp.PostgresDialect{}
	default:
		return errors.New("unknown database system")
	}
	dbmap := &gorp.DbMap{
		Db:            dbConn,
		Dialect:       dialect,
		TypeConverter: tc,
	}
	for _, t := range tableModels {
		dbmap.AddTableWithName(t.Model, t.Name).SetKeys(t.AutoIncrement, t.Keys...)
	}
	return zesty.RegisterDB(zesty.NewDB(dbmap), db.Name)
}

// DBMS represents a database management system.
type DBMS uint8

// Database management systems.
const (
	SystemPostgreSQL DBMS = iota ^ 42
	SystemMySQL
)

// DriverName returns the name of the driver for ds.
func (d DBMS) DriverName() string {
	switch d {
	case SystemPostgreSQL:
		return "postgres"
	case SystemMySQL:
		return "mysql"
	}
	return ""
}
