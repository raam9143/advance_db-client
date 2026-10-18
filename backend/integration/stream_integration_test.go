package integration
import(
 "bytes"
 "encoding/csv"
 "encoding/json"
 "strings"
 "testing"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
func TestSQLiteBoundedCSVStream(t *testing.T){
 s:=server.New("integration-test")
 code,body:=call(t,s,"POST","/v1/connections",server.ConnectionRequest{Name:"readonly-stream",Engine:"sqlite",Database:":memory:",ReadOnly:true})
 check(t,code,body,201)
 var c struct{ID string `json:"id"`}
 if err:=json.Unmarshal(body,&c);err!=nil{t.Fatal(err)}
 path:="/v1/connections/"+c.ID+"/stream"
 query:=server.QueryRequest{SQL:"WITH RECURSIVE data(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM data WHERE n<1205) SELECT n FROM data"}
 code,body=call(t,s,"POST",path,query);check(t,code,body,200)
 rows,err:=csv.NewReader(bytes.NewReader(body)).ReadAll()
 if err!=nil{t.Fatal(err)}
 if len(rows)!=1206||rows[1][0]!="1"||rows[len(rows)-1][0]!="1205"{t.Fatalf("bad streaming result: %d",len(rows))}
 code,body=call(t,s,"POST",path,server.QueryRequest{SQL:"SELECT '=2+2' AS cell"})
 check(t,code,body,200)
 if !strings.Contains(string(body),"'=2+2"){t.Fatal("CSV formula injection not neutralized")}
 for _,sql:=range []string{"DELETE FROM data","SELECT 1; DROP TABLE data","SELECT pg_sleep(10)"}{
  code,body=call(t,s,"POST",path,server.QueryRequest{SQL:sql});check(t,code,body,400)
 }
}
