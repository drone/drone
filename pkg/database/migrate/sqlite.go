package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/dchest/uniuri"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDriver MigrationDriver

func SQLite(tx *sql.Tx) Operation {
	return &SQLiteDriver{Tx: tx}
}

func (s *SQLiteDriver) CreateTable(tableName string, args []string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("CREATE TABLE %s (%s);", tableName, strings.Join(args, ", ")))
}

func (s *SQLiteDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tableName, newName))
}

func (s *SQLiteDriver) DropTable(tableName string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
}

func (s *SQLiteDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableName, columnSpec))
}

func (s *SQLiteDriver) DropColumns(tableName string, columnsToDrop []string) (sql.Result, error) {

	if len(columnsToDrop) == 0 {
		return nil, fmt.Errorf("No columns to drop.")
	}

	sql, err := s.getDDLFromTable(tableName)
	if err != nil {
		return nil, err
	}

	columns, err := fetchColumns(sql)
	if err != nil {
		return nil, err
	}

	columnNames := selectName(columns)

	var preparedColumns []string
	for k, column := range columnNames {
		listed := false
		for _, dropped := range columnsToDrop {
			if column == dropped {
				listed = true
				break
			}
		}
		if !listed {
			preparedColumns = append(preparedColumns, columns[k])
		}
	}

	if len(preparedColumns) == 0 {
		return nil, fmt.Errorf("No columns match, drops nothing.")
	}

	// Rename old table, here's our proxy
	proxyName := fmt.Sprintf("%s_%s", tableName, uniuri.NewLen(16))
	if result, err := s.RenameTable(tableName, proxyName); err != nil {
		return result, err
	}

	// Recreate table with dropped columns omitted
	if result, err := s.CreateTable(tableName, preparedColumns); err != nil {
		return result, err
	}

	// Move data from old table
	if result, err := s.Tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s;", tableName,
		strings.Join(selectName(preparedColumns), ", "), proxyName)); err != nil {
		return result, err
	}

	// Clean up proxy table
	return s.DropTable(proxyName)
}

func (s *SQLiteDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	sql, err := s.getDDLFromTable(tableName)
	if err != nil {
		return nil, err
	}

	columns, err := fetchColumns(sql)
	if err != nil {
		return nil, err
	}

	// We need a list of columns name to migrate data to the new table
	var oldColumnsName = selectName(columns)

	// newColumns will be used to create the new table
	var newColumns []string

	for k, column := range oldColumnsName {
		added := false
		for Old, New := range columnChanges {
			if column == Old {
				columnToAdd := strings.Replace(columns[k], Old, New, 1)
				newColumns = append(newColumns, columnToAdd)
				added = true
				break
			}
		}
		if !added {
			newColumns = append(newColumns, columns[k])
		}
	}

	// Rename current table
	proxyName := fmt.Sprintf("%s_%s", tableName, uniuri.NewLen(16))
	if result, err := s.RenameTable(tableName, proxyName); err != nil {
		return result, err
	}

	// Create new table with the new columns
	if result, err := s.CreateTable(tableName, newColumns); err != nil {
		return result, err
	}

	// Migrate data
	if result, err := s.Tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s", tableName,
		strings.Join(oldColumnsName, ", "), proxyName)); err != nil {
		return result, err
	}

	// Clean up proxy table
	return s.DropTable(proxyName)
}

func (s *SQLiteDriver) getDDLFromTable(tableName string) (string, error) {
	var sql string
	query := `SELECT sql FROM sqlite_master WHERE type='table' and name=?;`
	err := s.Tx.QueryRow(query, tableName).Scan(&sql)
	if err != nil {
		return "", err
	}
	return sql, nil
}
