package dbmigrate

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4/database"
	mysqldrv "github.com/golang-migrate/migrate/v4/database/mysql"
	postgresdrv "github.com/golang-migrate/migrate/v4/database/postgres"
	sqlite3drv "github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func openDriver(db *sql.DB, driverName string) (database.Driver, error) {
	switch driverName {
	case "mysql":
		return mysqldrv.WithInstance(db, &mysqldrv.Config{})
	case "postgres":
		return postgresdrv.WithInstance(db, &postgresdrv.Config{})
	case "sqlite", "sqlite3":
		return sqlite3drv.WithInstance(db, &sqlite3drv.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverName)
	}
}
