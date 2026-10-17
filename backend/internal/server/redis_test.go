package server
import(
 "bytes"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
 "github.com/raam9143/advance_db-client/backend/internal/redisclient"
)
func TestRedisOnlyInspectorEndpoint(t *testing.T){
 s:=New("secret")
 client,e:=redisclient.New(redisclient.Config{Host:"localhost",Port:6379,PlaintextLoopback:true});if e!=nil{t.Fatal(e)}
 id:=strings.Repeat("a",48)
 s.dbs[id]=&session{info:PublicConnection{ID:id,Engine:"redis",ReadOnly:true},owner:"service",redis:client,expiry:time.Now().Add(time.Hour)}
 for _,p:=range []string{"/query","/export","/explain","/table","/describe"}{
  req:=httptest.NewRequest("POST","/v1/connections/"+id+p,bytes.NewBufferString("{}"))
  req.Header.Set("Authorization","Bearer secret")
  w:=httptest.NewRecorder();s.ServeHTTP(w,req)
  if w.Code!=400{t.Fatalf("%s got %d",p,w.Code)}
 }
 req:=httptest.NewRequest("GET","/v1/connections/"+id+"/schema",nil)
 req.Header.Set("Authorization","Bearer secret")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=200||strings.TrimSpace(w.Body.String())!="[]"{t.Fatalf("schema got %d %s",w.Code,w.Body.String())}
}
func TestRedisConnectionTLSRequired(t *testing.T){
 s:=New("secret")
 body:=`{"name":"redis","engine":"redis","host":"127.0.0.1","port":6379,"database":"0","sslMode":"disable"}`
 req:=httptest.NewRequest("POST","/v1/connections",bytes.NewBufferString(body))
 req.Header.Set("Authorization","Bearer secret")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=400{t.Fatalf("plaintext accepted: %d %s",w.Code,w.Body.String())}
}
