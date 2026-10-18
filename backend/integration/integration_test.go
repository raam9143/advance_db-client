package integration

import (
 "bytes"
 "encoding/json"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "strings"
 "testing"
 _ "github.com/go-sql-driver/mysql"
 _ "github.com/jackc/pgx/v5/stdlib"
 _ "modernc.org/sqlite"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
func call(t *testing.T,h http.Handler,method,path string,body any)(int,[]byte){
 t.Helper();var raw []byte;if body!=nil{raw,_=json.Marshal(body)}
 req:=httptest.NewRequest(method,path,bytes.NewReader(raw));req.Header.Set("Authorization","Bearer integration-test")
 w:=httptest.NewRecorder();h.ServeHTTP(w,req);return w.Code,w.Body.Bytes()
}
func check(t *testing.T,status int,body []byte,want int){
 t.Helper();if status!=want{t.Fatalf("got %d want %d: %s",status,want,body)}
}
func runDatabase(t *testing.T,c server.ConnectionRequest){
 t.Helper();t.Setenv("ALLOW_WRITE_QUERIES","true")
 h:=server.New("integration-test")
 code,data:=call(t,h,"POST","/v1/connections",c);check(t,code,data,201)
 var info server.PublicConnection;if err:=json.Unmarshal(data,&info);err!=nil{t.Fatal(err)}
 path:="/v1/connections/"+info.ID
 table:="adc_integration_qa"
 for _,stmt:=range []string{"DROP TABLE IF EXISTS "+table,"CREATE TABLE "+table+" (id INTEGER PRIMARY KEY, name VARCHAR(50))","INSERT INTO "+table+" (id,name) VALUES (1,'first'),(2,'second'),(3,'third')"}{
  code,data=call(t,h,"POST",path+"/query",server.QueryRequest{SQL:stmt,ConfirmWrite:true});check(t,code,data,200)
 }
 t.Cleanup(func(){call(t,h,"POST",path+"/query",server.QueryRequest{SQL:"DROP TABLE IF EXISTS "+table,ConfirmWrite:true})})
 code,data=call(t,h,"GET",path+"/schema",nil);check(t,code,data,200)
 if !bytes.Contains(data,[]byte(table)){t.Fatalf("schema missing table: %s",data)}
 schema:=c.Database;if c.Engine=="sqlite"{schema="main"};if c.Engine=="postgres"||c.Engine=="cockroach"{schema="public"}
 q:="?schema="+schema+"&table="+table
 code,data=call(t,h,"GET",path+"/describe"+q,nil);check(t,code,data,200)
 var d server.TableDescription;if err:=json.Unmarshal(data,&d);err!=nil{t.Fatal(err)}
 if len(d.Columns)!=2||!d.Columns[0].PrimaryKey{t.Fatalf("invalid column metadata: %+v",d)}
 code,data=call(t,h,"GET",path+"/table"+q+"&limit=2",nil);check(t,code,data,200)
 var page server.PageResult;if err:=json.Unmarshal(data,&page);err!=nil{t.Fatal(err)}
 if page.RowCount!=2||page.NextOffset==nil||*page.NextOffset!=2{t.Fatalf("invalid page: %+v",page)}
 code,data=call(t,h,"GET",path+"/table"+q+"&limit=2&offset=2",nil);check(t,code,data,200)
 page=server.PageResult{};if err:=json.Unmarshal(data,&page);err!=nil{t.Fatal(err)}
 if page.RowCount!=1||page.NextOffset!=nil{t.Fatalf("invalid final page: %+v",page)}
 code,data=call(t,h,"POST",path+"/explain",server.QueryRequest{SQL:"SELECT 1"});check(t,code,data,200)
 code,data=call(t,h,"POST",path+"/export?format=json",server.QueryRequest{SQL:"SELECT 1"});check(t,code,data,200)
 if !bytes.Contains(data,[]byte("columns")){t.Fatalf("JSON export malformed: %s",data)}
 code,data=call(t,h,"POST",path+"/export",server.QueryRequest{SQL:"SELECT '=2+2' AS f"});check(t,code,data,200)
 if !strings.Contains(string(data),"'=2+2"){t.Fatalf("CSV formula escaped incorrectly: %s",data)}
}
func TestSQLite(t *testing.T){runDatabase(t,server.ConnectionRequest{Name:"sqlite",Engine:"sqlite",Database:":memory:"})}
func TestPostgres(t *testing.T){
 if os.Getenv("TEST_POSTGRES_HOST")==""{t.Skip("requires TEST_POSTGRES_HOST")}
 runDatabase(t,server.ConnectionRequest{Name:"pg",Engine:"postgres",Host:os.Getenv("TEST_POSTGRES_HOST"),Port:5432,User:"adb",Password:"adbtest",Database:"adb",SSLMode:"disable"})
}
func TestMySQL(t *testing.T){
 if os.Getenv("TEST_MYSQL_HOST")==""{t.Skip("requires TEST_MYSQL_HOST")}
 runDatabase(t,server.ConnectionRequest{Name:"mysql",Engine:"mysql",Host:os.Getenv("TEST_MYSQL_HOST"),Port:3306,User:"adb",Password:"adbtest",Database:"adb",SSLMode:"false"})
}
func TestEngineAlias(t *testing.T){
 for _,tc:=range []struct{engine,want string}{{"cockroach","pgx"},{"mariadb","mysql"}}{
  driver,_,err:=server.ConfigDSN(server.ConnectionRequest{Name:"engine-test",Engine:tc.engine,Host:"localhost",Port:5432,Database:"adb",User:"x",Password:"secret",SSLMode:map[string]string{"cockroach":"disable","mariadb":"false"}[tc.engine]})
  if err!=nil||driver!=tc.want{t.Fatal(fmt.Sprint(tc,driver,err))}
 }
}

// Future-version compatibility tests run only against disposable CI databases.
// A version becoming available does not imply production support until this
// suite and an explicit reviewed driver update have succeeded.
func TestMariaDBLive(t *testing.T){
 host:=os.Getenv("TEST_MARIADB_HOST")
 if host==""{t.Skip("requires disposable MariaDB host")}
 runDatabase(t,server.ConnectionRequest{Name:"mariadb-compat",Engine:"mariadb",Host:host,Port:3306,User:"adb",Password:"adbtest",Database:"adb",SSLMode:"false"})
}
func TestCockroachLive(t *testing.T){
 host:=os.Getenv("TEST_COCKROACH_HOST")
 if host==""{t.Skip("requires disposable CockroachDB host")}
 runDatabase(t,server.ConnectionRequest{Name:"cockroach-compat",Engine:"cockroach",Host:host,Port:26257,User:"root",Database:"defaultdb",SSLMode:"disable"})
}
