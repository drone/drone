package migrate

import (
	"database/sql"
	"os"
	"testing"

	"github.com/russross/meddler"
)

type Sample struct {
	ID   int64  `meddler:"id,pk"`
	Imel string `meddler:"imel"`
	Name string `meddler:"name"`
}

type RenameSample struct {
	ID    int64  `meddler:"id,pk"`
	Email string `meddler:"email"`
	Name  string `meddler:"name"`
}

type AddColumnSample struct {
	ID   int64  `meddler:"id,pk"`
	Imel string `meddler:"imel"`
	Name string `meddler:"name"`
	Url  string `meddler:"url"`
	Num  int64  `meddler:"num"`
}

// ---------- revision 1

type revision1 struct{}

func (r *revision1) Up(op Operation) error {
	_, err := op.CreateTable("samples", []string{
		"id INTEGER PRIMARY KEY AUTOINCREMENT",
		"imel VARCHAR(255) UNIQUE",
		"name VARCHAR(255)",
	})
	return err
}

func (r *revision1) Down(op Operation) error {
	_, err := op.DropTable("samples")
	return err
}

func (r *revision1) Revision() int64 {
	return 1
}

// ---------- end of revision 1

// ---------- revision 2

type revision2 struct{}

func (r *revision2) Up(op Operation) error {
	_, err := op.RenameTable("samples", "examples")
	return err
}

func (r *revision2) Down(op Operation) error {
	_, err := op.RenameTable("examples", "samples")
	return err
}

func (r *revision2) Revision() int64 {
	return 2
}

// ---------- end of revision 2

// ---------- revision 3

type revision3 struct{}

func (r *revision3) Up(op Operation) error {
	if _, err := op.AddColumn("samples", "url VARCHAR(255)"); err != nil {
		return err
	}
	_, err := op.AddColumn("samples", "num INTEGER")
	return err
}

func (r *revision3) Down(op Operation) error {
	_, err := op.DropColumns("samples", []string{"num", "url"})
	return err
}

func (r *revision3) Revision() int64 {
	return 3
}

// ---------- end of revision 3

// ---------- revision 4

type revision4 struct{}

func (r *revision4) Up(op Operation) error {
	_, err := op.RenameColumns("samples", map[string]string{
		"imel": "email",
	})
	return err
}

func (r *revision4) Down(op Operation) error {
	_, err := op.RenameColumns("samples", map[string]string{
		"email": "imel",
	})
	return err
}

func (r *revision4) Revision() int64 {
	return 4
}

// ----------

var db *sql.DB

var testSchema = `
CREATE TABLE samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	imel VARCHAR(255) UNIQUE,
	name VARCHAR(255)
);
`

var dataDump = []string{
	`INSERT INTO samples (imel, name) VALUES ('test@example.com', 'Test Tester');`,
	`INSERT INTO samples (imel, name) VALUES ('foo@bar.com', 'Foo Bar');`,
	`INSERT INTO samples (imel, name) VALUES ('crash@bandicoot.io', 'Crash Bandicoot');`,
}

func TestMigrateCreateTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	Driver = SQLite

	mgr := New(db)
	if err := mgr.Add(&revision1{}).Migrate(); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	sample := Sample{
		ID:   1,
		Imel: "test@example.com",
		Name: "Test Tester",
	}
	if err := meddler.Save(db, "samples", &sample); err != nil {
		t.Errorf("Can not save data: %q", err)
	}
}

func TestMigrateRenameTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	Driver = SQLite

	mgr := New(db)
	if err := mgr.Add(&revision1{}).Migrate(); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	loadFixture(t)

	if err := mgr.Add(&revision2{}).Migrate(); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	sample := Sample{}
	if err := meddler.QueryRow(db, &sample, `SELECT * FROM examples WHERE id = ?`, 2); err != nil {
		t.Errorf("Can not fetch data: %q", err)
	}

	if sample.Imel != "foo@bar.com" {
		t.Errorf("Column doesn't match. Expect: %s, got: %s", "foo@bar.com", sample.Imel)
	}
}

type TableInfo struct {
	CID       int64       `meddler:"cid,pk"`
	Name      string      `meddler:"name"`
	Type      string      `meddler:"type"`
	Notnull   bool        `meddler:"notnull"`
	DfltValue interface{} `meddler:"dflt_value"`
	PK        bool        `meddler:"pk"`
}

func TestMigrateAddRemoveColumns(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	Driver = SQLite

	mgr := New(db)
	if err := mgr.Add(&revision1{}, &revision3{}).Migrate(); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	var columns []*TableInfo
	if err := meddler.QueryAll(db, &columns, `PRAGMA table_info(samples);`); err != nil {
		t.Errorf("Can not access table info: %q", err)
	}

	if len(columns) < 5 {
		t.Errorf("Expect length columns: %d\nGot: %d", 5, len(columns))
	}

	var row = AddColumnSample{
		ID:   33,
		Name: "Foo",
		Imel: "foo@bar.com",
		Url:  "http://example.com",
		Num:  42,
	}
	if err := meddler.Save(db, "samples", &row); err != nil {
		t.Errorf("Can not save into database: %q", err)
	}

	if err := mgr.MigrateTo(1); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	var another_columns []*TableInfo
	if err := meddler.QueryAll(db, &another_columns, `PRAGMA table_info(samples);`); err != nil {
		t.Errorf("Can not access table info: %q", err)
	}

	if len(another_columns) != 3 {
		t.Errorf("Expect length columns = %d, got: %d", 3, len(columns))
	}
}

func TestRenameColumn(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	Driver = SQLite

	mgr := New(db)
	if err := mgr.Add(&revision1{}, &revision4{}).MigrateTo(1); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	loadFixture(t)

	if err := mgr.MigrateTo(4); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	row := RenameSample{}
	if err := meddler.QueryRow(db, &row, `SELECT * FROM samples WHERE id = 3;`); err != nil {
		t.Errorf("Can not query database: %q", err)
	}

	if row.Email != "crash@bandicoot.io" {
		t.Errorf("Expect %s, got %s", "crash@bandicoot.io", row.Email)
	}
}

func TestMigrateExistingTable(t *testing.T) {
	defer tearDown()
	if err := setUp(); err != nil {
		t.Fatalf("Error preparing database: %q", err)
	}

	Driver = SQLite

	if _, err := db.Exec(testSchema); err != nil {
		t.Errorf("Can not create database: %q", err)
	}

	loadFixture(t)

	mgr := New(db)
	if err := mgr.Add(&revision4{}).Migrate(); err != nil {
		t.Errorf("Can not migrate: %q", err)
	}

	var rows []*RenameSample
	if err := meddler.QueryAll(db, &rows, `SELECT * from samples;`); err != nil {
		t.Errorf("Can not query database: %q", err)
	}

	if len(rows) != 3 {
		t.Errorf("Expect rows length = %d, got %d", 3, len(rows))
	}

	if rows[1].Email != "foo@bar.com" {
		t.Errorf("Expect email = %s, got %s", "foo@bar.com", rows[1].Email)
	}
}

func setUp() error {
	var err error
	db, err = sql.Open("sqlite3", "migration_tests.sqlite")
	return err
}

func tearDown() {
	db.Close()
	os.Remove("migration_tests.sqlite")
}

func loadFixture(t *testing.T) {
	for _, sql := range dataDump {
		if _, err := db.Exec(sql); err != nil {
			t.Errorf("Can not insert into database: %q", err)
		}
	}
}
