package integration
import(
 "encoding/json"
 "os"
 "strconv"
 "testing"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
// Optional smoke tests against an explicitly provisioned trusted Oracle TCPS test server.
// No Oracle server or wallet is provisioned in default public CI.
func TestOracleTCPSLive(t *testing.T){
 host,service,user,password:=os.Getenv("TEST_ORACLE_HOST"),os.Getenv("TEST_ORACLE_SERVICE"),os.Getenv("TEST_ORACLE_USER"),os.Getenv("TEST_ORACLE_PASSWORD")
 if host==""||service==""||user==""||password==""{t.Skip("set TEST_ORACLE_HOST, TEST_ORACLE_SERVICE, TEST_ORACLE_USER, TEST_ORACLE_PASSWORD to run")}
 port:=2484;if v:=os.Getenv("TEST_ORACLE_PORT");v!=""{n,e:=strconv.Atoi(v);if e!=nil{t.Fatal(e)};port=n}
 mode:=os.Getenv("TEST_ORACLE_SERVICE_MODE");if mode==""{mode="service"}
 h:=server.New("integration-test")
 c:=server.ConnectionRequest{Name:"oracle-ci",Engine:"oracle",Host:host,Port:port,Database:service,User:user,Password:password,ServiceMode:mode,SSLMode:"verify-full",ReadOnly:true}
 status,body:=call(t,h,"POST","/v1/connections",c);check(t,status,body,201)
 var info server.PublicConnection
 if err:=json.Unmarshal(body,&info);err!=nil{t.Fatal(err)}
 if !info.ReadOnly||info.Engine!="oracle"{t.Fatalf("unsafe Oracle session %+v",info)}
 path:="/v1/connections/"+info.ID
 status,body=call(t,h,"POST",path+"/query",server.QueryRequest{SQL:"SELECT 1 AS ONE FROM dual"})
 check(t,status,body,200)
 var result server.QueryResult
 if err:=json.Unmarshal(body,&result);err!=nil||result.RowCount!=1{t.Fatalf("Oracle smoke query failed: %v %s",err,body)}
 status,body=call(t,h,"GET",path+"/schema",nil);check(t,status,body,200)
 status,body=call(t,h,"POST",path+"/query",server.QueryRequest{SQL:"DELETE FROM dual"})
 check(t,status,body,400)
}
