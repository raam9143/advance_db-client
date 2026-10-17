package server
import(
 "bytes"
 "net/http/httptest"
 "testing"
)
func TestSSHPolicyDenyByDefault(t *testing.T){
 t.Setenv("SSH_TUNNEL_ENABLED","false")
 s:=New("token")
 req:=httptest.NewRequest("POST","/v1/connections",bytes.NewBufferString(`{"name":"test","engine":"postgres","host":"127.0.0.1","port":5432,"database":"sample","user":"readonly","password":"x","sslMode":"disable","tunnel":{"host":"ssh.local","port":22,"user":"sshuser","privateKey":"invalid"}}`))
 req.Header.Set("Authorization","Bearer token")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=400{t.Fatalf("SSH policy deny got %d %s",w.Code,w.Body.String())}
}
func TestSSHDenyUnsupportedEngines(t *testing.T){
 s:=New("token")
 req:=httptest.NewRequest("POST","/v1/connections",bytes.NewBufferString(`{"name":"test","engine":"oracle","host":"127.0.0.1","port":2484,"database":"XE","user":"user","password":"x","sslMode":"require","tunnel":{"host":"ssh.local","port":22,"user":"u","privateKey":"dummy"}}`))
 req.Header.Set("Authorization","Bearer token")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=400{t.Fatalf("unsupported SSH driver got %d",w.Code)}
}
