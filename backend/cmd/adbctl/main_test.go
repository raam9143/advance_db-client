package main
import("bytes";"context";"net/http";"net/http/httptest";"strings";"testing")
func TestRejectWrites(t *testing.T){
 env:=func(k string)string{return "test"}
 err:=run(context.Background(),[]string{"query","-id",strings.Repeat("a",48),"-sql","DROP TABLE data"},&bytes.Buffer{},env)
 if err==nil{t.Fatal("write query accepted")}
}
func TestQueryRequest(t *testing.T){
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method!="POST"||r.Header.Get("Authorization")!="Bearer secret"||!strings.HasSuffix(r.URL.Path,"/query"){t.Errorf("bad request: %s",r.URL)}
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte(`{"columns":["x"],"rows":[[1]]}`))
 }))
 defer srv.Close()
 env:=func(k string)string{if k=="ADB_API_TOKEN"{return "secret"};return srv.URL}
 var out bytes.Buffer
 err:=run(context.Background(),[]string{"query","-id",strings.Repeat("a",48),"-sql","SELECT ?","-params","[1]"},&out,env)
 if err!=nil||!strings.Contains(out.String(),"columns"){t.Fatalf("%s %v",out.String(),err)}
}
func TestMissingToken(t *testing.T){
 err:=run(context.Background(),[]string{"connections"},&bytes.Buffer{},func(string)string{return ""})
 if err==nil{t.Fatal("missing API token allowed")}
}
