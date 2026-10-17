package integration

import(
 "encoding/json"
 _ "github.com/microsoft/go-mssqldb"
 "os"
 "strings"
 "testing"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
func TestSQLServerLive(t *testing.T){
 host:=os.Getenv("TEST_MSSQL_HOST")
 if host==""{t.Skip("requires an isolated SQL Server test instance")}
 t.Setenv("SQLSERVER_ALLOW_UNVERIFIED_LOOPBACK","true")
 t.Setenv("ALLOW_WRITE_QUERIES","true")
 h:=server.New("integration-test")
 c:=server.ConnectionRequest{Name:"mssql-ci",Engine:"sqlserver",Host:host,Port:1433,Database:"master",User:"sa",Password:os.Getenv("TEST_MSSQL_PASSWORD"),SSLMode:"local-insecure",ReadOnly:false}
 status,body:=call(t,h,"POST","/v1/connections",c);check(t,status,body,201)
 var info server.PublicConnection
 if e:=json.Unmarshal(body,&info);e!=nil{t.Fatal(e)}
 endpoint:="/v1/connections/"+info.ID
 name:="adb_sqlserver_qa"
 defer call(t,h,"POST",endpoint+"/query",server.QueryRequest{SQL:"DROP TABLE IF EXISTS dbo."+name,ConfirmWrite:true})
 for _,q:=range []string{
  "DROP TABLE IF EXISTS dbo."+name,
  "CREATE TABLE dbo."+name+" (id INT NOT NULL PRIMARY KEY, name NVARCHAR(50) NOT NULL)",
  "INSERT INTO dbo."+name+" (id,name) VALUES (1,'first'),(2,'second'),(3,'third')",
 }{
  status,body=call(t,h,"POST",endpoint+"/query",server.QueryRequest{SQL:q,ConfirmWrite:true})
  check(t,status,body,200)
 }
 status,body=call(t,h,"GET",endpoint+"/schema",nil);check(t,status,body,200)
 if !strings.Contains(string(body),name){t.Fatalf("schema missing table: %s",body)}
 p:="?schema=dbo&table="+name
 status,body=call(t,h,"GET",endpoint+"/describe"+p,nil);check(t,status,body,200)
 var d server.TableDescription
 if err:=json.Unmarshal(body,&d);err!=nil{t.Fatal(err)}
 if len(d.Columns)!=2||!d.Columns[0].PrimaryKey{t.Fatalf("bad SQL Server metadata: %+v",d)}
 status,body=call(t,h,"GET",endpoint+"/table"+p+"&limit=2",nil);check(t,status,body,200)
 var page server.PageResult
 if err:=json.Unmarshal(body,&page);err!=nil{t.Fatal(err)}
 if page.RowCount!=2||page.NextOffset==nil{t.Fatalf("incorrect SQL Server pagination: %+v",page)}
 status,body=call(t,h,"POST",endpoint+"/query",server.QueryRequest{SQL:"SELECT name FROM dbo."+name+" WHERE id=@p1",Parameters:[]any{float64(1)}});check(t,status,body,200)
 if !strings.Contains(string(body),"first"){t.Fatalf("SQL Server bind parameters failed: %s",body)}
}
