package server

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequireToken(t *testing.T) {
	s := New("correct")
	r := httptest.NewRequest("GET", "/v1/connections", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
	r.Header.Set("Authorization", "Bearer correct")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
func TestReadOnlySQL(t *testing.T) {
	cases := []struct {
		sql  string
		want bool
	}{{"SELECT 1", true}, {"SELECT * FROM public.foo;", true}, {"  WITH x AS (SELECT 1) SELECT * FROM x", true}, {"SELECT 1; DELETE FROM users", false}, {"SELECT 1 --comment", false}, {"DELETE FROM x", false}, {"SELECT 1 FOR UPDATE", false}, {"SELECT * INTO new_table FROM old_table", false}, {"SELECT NEXT VALUE FOR seq", false}, {"EXPLAIN DELETE FROM users", false}, {"SELECT 1 /*abc*/", false}, {"", false}}
	for _, v := range cases {
		if got := ReadOnlySQL(v.sql); got != v.want {
			t.Errorf("%q: got %v want %v", v.sql, got, v.want)
		}
	}
}
func TestConfigDSN(t *testing.T) {
	d, dsn, err := ConfigDSN(ConnectionRequest{Name: "local", Engine: "postgres", Host: "db.local", Port: 5432, Database: "test", User: "user", Password: "p@ss", SSLMode: "require"})
	if err != nil || d != "pgx" || !strings.Contains(dsn, "p%40ss") {
		t.Fatalf("%s %s %v", d, dsn, err)
	}
	_, _, err = ConfigDSN(ConnectionRequest{Name: "bad", Engine: "oracle"})
	if err == nil {
		t.Fatal("oracle with missing parameters must fail")
	}
}
func TestForbiddenPath(t *testing.T) {
	s := New("correct")
	r := httptest.NewRequest("GET", "/v1/no-such", nil)
	r.Header.Set("Authorization", "Bearer correct")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("got %d", w.Code)
	}
}

func TestSQLiteSandbox(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	t.Setenv("SQLITE_ALLOWED_DIR", root)
	good := filepath.Join(root, "inside.db")
	if err := os.WriteFile(good, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	driver, dsn, err := ConfigDSN(ConnectionRequest{Name: "local", Engine: "sqlite", Database: good, ReadOnly: true})
	if err != nil || driver != "sqlite" || !strings.Contains(dsn, "mode=ro") {
		t.Fatalf("SQLite: %s %s %v", driver, dsn, err)
	}
	forbidden := filepath.Join(outside, "outside.db")
	if err := os.WriteFile(forbidden, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ConfigDSN(ConnectionRequest{Name: "bad", Engine: "sqlite", Database: forbidden}); err == nil {
		t.Fatal("outside path accepted")
	}
	link := filepath.Join(root, "symlink.db")
	if err = os.Symlink(forbidden, link); err == nil {
		if _, _, err = ConfigDSN(ConnectionRequest{Name: "bad", Engine: "sqlite", Database: link}); err == nil {
			t.Fatal("escaping symlink accepted")
		}
	}
}

type mockDriver struct{}
type mockConn struct{}
type mockTx struct{}
type mockRows struct{ n int }

func (mockDriver) Open(string) (driver.Conn, error)                           { return mockConn{}, nil }
func (mockConn) Prepare(string) (driver.Stmt, error)                          { return nil, errors.New("not implemented") }
func (mockConn) Close() error                                                 { return nil }
func (mockConn) Begin() (driver.Tx, error)                                    { return mockTx{}, nil }
func (mockConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) { return mockTx{}, nil }
func (mockConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &mockRows{}, nil
}
func (mockTx) Commit() error       { return nil }
func (mockTx) Rollback() error     { return nil }
func (mockRows) Columns() []string { return []string{"value"} }
func (mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.n == 2 {
		return io.EOF
	}
	r.n++
	dest[0] = int64(r.n)
	return nil
}
func TestQueryReadOnlyTransaction(t *testing.T) {
	sql.Register("adb-mock-test", mockDriver{})
	db, e := sql.Open("adb-mock-test", "")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New("token")
	res, e := s.run(context.Background(), &session{info: PublicConnection{Engine: "postgres", ReadOnly: true}, db: db}, QueryRequest{SQL: "SELECT 1", TimeoutSeconds: 2})
	if e != nil || res.RowCount != 2 || len(res.Columns) != 1 || res.Rows[1][0] != int64(2) {
		t.Fatalf("unexpected result: %+v %v", res, e)
	}
	if _, e = s.run(context.Background(), &session{info: PublicConnection{ReadOnly: true}, db: db}, QueryRequest{SQL: "DROP TABLE users"}); e == nil {
		t.Fatal("read-only write accepted")
	}
}
