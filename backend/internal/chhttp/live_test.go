package chhttp

import(
 "context"
 "database/sql/driver"
 "fmt"
 "io"
 "net/http"
 "net/http/httptest"
 "net/url"
 "os"
 "strings"
 "testing"
 "time"
)
// A real ClickHouse HTTP service is tunneled behind a TLS test proxy. The tested
// adapter still authenticates and verifies TLS, but native ClickHouse TLS cert
// provisioning and production-hostname verification are separate acceptance work.
func TestLiveClickHouseHTTPSProxy(t *testing.T){
 host:=os.Getenv("TEST_CLICKHOUSE_HOST")
 if host==""{t.Skip("requires TEST_CLICKHOUSE_HOST")}
 upstream:="http://"+host+":8123/"
 proxy:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  u,p,ok:=r.BasicAuth()
  if !ok||u!="alice"||p!="unit-secret"{http.Error(w,"bad proxy credentials",401);return}
  target,err:=url.Parse(upstream)
  if err!=nil{t.Error(err);http.Error(w,"bad upstream",500);return}
  target.RawQuery=r.URL.RawQuery
  body,err:=io.ReadAll(io.LimitReader(r.Body,100000))
  if err!=nil{http.Error(w,"bad request",400);return}
  ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
  req,err:=http.NewRequestWithContext(ctx,"POST",target.String(),strings.NewReader(string(body)))
  if err!=nil{http.Error(w,"bad upstream request",500);return}
  req.Header.Set("X-ClickHouse-User","adb")
  req.Header.Set("X-ClickHouse-Key","adbtest")
  resp,err:=(&http.Client{Timeout:15*time.Second}).Do(req)
  if err!=nil{http.Error(w,"upstream unavailable",502);return}
  defer resp.Body.Close()
  w.WriteHeader(resp.StatusCode)
  _,_=io.Copy(w,io.LimitReader(resp.Body,8<<20))
 }))
 defer proxy.Close()
 u,_:=url.Parse(strings.Replace(proxy.URL,"https://","https://alice:unit-secret@",1)+"/?database=default")
 c:=&conn{endpoint:u,client:proxy.Client()}
 if err:=c.Ping(context.Background());err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{
  query string;args []driver.NamedValue;expected string
 }{
  {"SELECT number AS value FROM system.numbers LIMIT 2",nil,"0"},
  {"SELECT {p1:UInt8} AS value",[]driver.NamedValue{{Ordinal:1,Value:int64(7)}},"7"},
  {"SELECT database,name FROM system.tables WHERE database=currentDatabase() AND is_temporary=0 ORDER BY database,name LIMIT 1",nil,""},
 }{
  result,err:=c.QueryContext(context.Background(),tc.query,tc.args)
  if err!=nil{t.Fatalf("real ClickHouse query %q: %v",tc.query,err)}
  columns:=result.Columns()
  if len(columns)==0{t.Fatalf("%s missing metadata columns",tc.query)}
  values:=make([]driver.Value,len(columns))
  if err:=result.Next(values);err!=nil&&err!=io.EOF{t.Fatal(err)}
  if tc.expected!=""&&fmt.Sprint(values[0])!=tc.expected{t.Fatalf("%s: got %v want %s",tc.query,values[0],tc.expected)}
  _=result.Close()
 }
 for _,unsafe:=range []string{"DROP TABLE something","SELECT 1 FORMAT Native","SELECT 1 SETTINGS readonly=0"}{
  if _,err:=c.QueryContext(context.Background(),unsafe,nil);err==nil{t.Fatalf("unsafe ClickHouse SQL accepted: %s",unsafe)}
 }
}
