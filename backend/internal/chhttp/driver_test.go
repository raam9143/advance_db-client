package chhttp
import("context";"database/sql/driver";"net/http";"net/http/httptest";"net/url";"strings";"testing")
func TestInvalidEndpoint(t *testing.T){for _,s:=range []string{"http://u:p@example.com:8123/?database=d","https://u:p@example.com/?database=d","https://u:p@example.com:8443/?database=d&readonly=0"}{if _,e:=(&Driver{}).Open(s);e==nil{t.Error("allowed insecure DSN")}}}
func TestReadOnlyHTTP(t *testing.T){
 calls:=0
 server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  calls++;u,p,ok:=r.BasicAuth();if !ok||u!="alice"||p!="private"{t.Error("bad basic auth")}
  v:=r.URL.Query();if r.Method!="POST"||v.Get("readonly")!="2"||v.Get("default_format")!="JSON"||v.Get("database")!="analytics"{t.Errorf("unexpected request %v",v)}
  if calls==2&&v.Get("param_p1")!="1"{t.Error("missing bound param")}
  _,_=w.Write([]byte(`{"meta":[{"name":"value","type":"UInt8"}],"data":[{"value":1}],"rows":1}`))
 }))
 defer server.Close()
 u,_:=url.Parse(strings.Replace(server.URL,"https://","https://alice:private@",1)+"/?database=analytics")
 c:=&conn{u,server.Client()}
 r,e:=c.QueryContext(context.Background(),"SELECT 1",nil);if e!=nil{t.Fatal(e)}
 out:=make([]driver.Value,1);if e=r.Next(out);e!=nil||out[0]!=int64(1){t.Errorf("invalid rows: %v %v",out,e)}
 if _,e=c.QueryContext(context.Background(),"SELECT {p1:UInt8}",[]driver.NamedValue{{Ordinal:1,Value:int64(1)}});e!=nil{t.Fatal(e)}
 for _,sql:=range []string{"DROP TABLE x","SELECT 1; DROP TABLE x","SELECT 1 FORMAT Native","SELECT 1 SETTINGS readonly=0"}{if _,e:=c.QueryContext(context.Background(),sql,nil);e==nil{t.Errorf("allowed %s",sql)}}
 if _,e=c.BeginTx(context.Background(),driver.TxOptions{ReadOnly:false});e==nil{t.Error("writable transaction accepted")}
 if _,e=c.ExecContext(context.Background(),"DROP TABLE t",nil);e==nil{t.Error("writable SQL accepted")}
}
