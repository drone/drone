package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/dchest/uniuri"
	_ "github.com/mattn/go-sqlite3"
)

type sqliteDriver struct {
	Tx *sql.Tx
}

func SQLite(tx *sql.Tx) *MigrationDriver {
	return &MigrationDriver{
		Tx:        tx,
		Operation: &sqliteDriver{Tx: tx},
	}
}

func (s *sqliteDriver) CreateTable(tableName string, args []string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("CREATE TABLE %s (%s);", tableName, strings.Join(args, ", ")))
}

func (s *sqliteDriver) RenameTable(tableName, newName string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tableName, newName))
}

func (s *sqliteDriver) DropTable(tableName string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
}

func (s *sqliteDriver) AddColumn(tableName, columnSpec string) (sql.Result, error) {
	return s.Tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableName, columnSpec))
}

func (s *sqliteDriver) DropColumns(tableName string, columnsToDrop []string) (sql.Result, error) {
	var err error
	var result sql.Result

	if len(columnsToDrop) == 0 {
		return nil, fmt.Errorf("No columns to drop.")
	}

	tableSQL, err := s.getDDLFromTable(tableName)
	if err != nil {
		return nil, err
	}

	columns, err := fetchColumns(tableSQL)
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

	// fetch indices for this table
	oldSQLIndices, err := s.getDDLFromIndex(tableName)
	if err != nil {
		return nil, err
	}

	var oldIdxColumns [][]string
	for _, idx := range oldSQLIndices {
		idxCols, err := fetchColumns(idx)
		if err != nil {
			return nil, err
		}
		oldIdxColumns = append(oldIdxColumns, idxCols)
	}

	var indices []string
	for k, idx := range oldSQLIndices {
		listed := false
	OIdxLoop:
		for _, oidx := range oldIdxColumns[k] {
			for _, cols := range columnsToDrop {
				if oidx == cols {
					listed = true
					break OIdxLoop
				}
			}
		}
		if !listed {
			indices = append(indices, idx)
		}
	}

	// Rename old table, here's our proxy
	proxyName := fmt.Sprintf("%s_%s", tableName, uniuri.NewLen(16))
	if result, err := s.RenameTable(tableName, proxyName); err != nil {
		return result, err
	}

	// Recreate table with dropped columns omitted
	if result, err = s.CreateTable(tableName, preparedColumns); err != nil {
		return result, err
	}

	// Move data from old table
	if result, err = s.Tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s;", tableName,
		strings.Join(selectName(preparedColumns), ", "), proxyName)); err != nil {
		return result, err
	}

	// Clean up proxy table
	if result, err = s.DropTable(proxyName); err != nil {
		return result, err
	}

	// Recreate Indices
	for _, idx := range indices {
		if result, err = s.Tx.Exec(idx); err != nil {
			return result, err
		}
	}
	return result, err
}

func (s *sqliteDriver) RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error) {
	var err error
	var result sql.Result

	tableSQL, err := s.getDDLFromTable(tableName)
	if err != nil {
		return nil, err
	}

	columns, err := fetchColumns(tableSQL)
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

	// fetch indices for this table
	oldSQLIndices, err := s.getDDLFromIndex(tableName)
	if err != nil {
		return nil, err
	}

	var idxColumns [][]string
	for _, idx := range oldSQLIndices {
		idxCols, err := fetchColumns(idx)
		if err != nil {
			return nil, err
		}
		idxColumns = append(idxColumns, idxCols)
	}

	var indices []string
	for k, idx := range oldSQLIndices {
		added := false
	IdcLoop:
		for _, oldIdx := range idxColumns[k] {
			for Old, New := range columnChanges {
				if oldIdx == Old {
					indx := strings.Replace(idx, Old, New, 2)
					indices = append(indices, indx)
					added = true
					break IdcLoop
				}
			}
		}
		if !added {
			indices = append(indices, idx)
		}
	}

	// Rename current table
	proxyName := fmt.Sprintf("%s_%s", tableName, uniuri.NewLen(16))
	if result, err := s.RenameTable(tableName, proxyName); err != nil {
		return result, err
	}

	// Create new table with the new columns
	if result, err = s.CreateTable(tableName, newColumns); err != nil {
		return result, err
	}

	// Migrate data
	if result, err = s.Tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s", tableName,
		strings.Join(oldColumnsName, ", "), proxyName)); err != nil {
		return result, err
	}

	// Clean up proxy table
	if result, err = s.DropTable(proxyName); err != nil {
		return result, err
	}

	for _, idx := range indices {
		if result, err = s.Tx.Exec(idx); err != nil {
			return result, err
		}
	}
	return result, err
}

func (s *sqliteDriver) getDDLFromTable(tableName string) (string, error) {
	var sql string
	query := `SELECT sql FROM sqlite_master WHERE type='table' and name=?;`
	err := s.Tx.QueryRow(query, tableName).Scan(&sql)
	if err != nil {
		return "", err
	}
	return sql, nil
}

func (s *sqliteDriver) getDDLFromIndex(tableName string) ([]string, error) {
	var sqls []string

	query := `SELECT sql FROM sqlite_master WHERE type='index' and tbl_name=?;`
	rows, err := s.Tx.Query(query, tableName)
	if err != nil {
		return sqls, err
	}

	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			// This error came from autoindex, since its sql value is null,
			// we want to continue.
			if strings.Contains(err.Error(), "Scan pair: <nil> -> *string") {
				continue
			}
			return sqls, err
		}
		sqls = append(sqls, sql)
	}

	if err := rows.Err(); err != nil {
		return sqls, err
	}

	return sqls, nil
}
