package server
import(
 "bytes"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)
func TestMongoRejectNonTLSRemote(t *testing.T){
 t.Setenv("MONGO_ALLOW_PLAINTEXT_LOOPBACK","true")
 s:=New("secret")
 req:=httptest.NewRequest("POST","/v1/connections",bytes.NewBufferString(`{"name":"unsafe","engine":"mongo","host":"remote.example.org","port":27017,"database":"demo","sslMode":"disable"}`))
 req.Header.Set("Authorization","Bearer secret")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=400{t.Fatalf("insecure MongoDB connection admitted: %d %s",w.Code,w.Body.String())}
}
func TestMongoSQLRouteGuard(t *testing.T){
 s:=New("secret")
 id:=strings.Repeat("d",48)
 s.dbs[id]=&session{owner:"service",info:PublicConnection{Engine:"mongo",ID:id,ReadOnly:true},expiry:time.Now().Add(time.Hour)}
 for _,methodAndPath:=range []struct{method,path string}{
  {"POST","query"},{"POST","export"},{"POST","explain"},{"GET","describe"},{"GET","table"},{"POST","redis"},
 }{
  req:=httptest.NewRequest(methodAndPath.method,"/v1/connections/"+id+"/"+methodAndPath.path,bytes.NewBufferString("{}"))
  req.Header.Set("Authorization","Bearer secret")
  w:=httptest.NewRecorder();s.ServeHTTP(w,req)
  if w.Code!=400{t.Fatalf("%s got %d instead of guarded rejection",methodAndPath.path,w.Code)}
 }
}
