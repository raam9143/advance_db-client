package server
import(
 "bytes"
 "crypto/rand"
 "encoding/base64"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "github.com/raam9143/advance_db-client/backend/internal/vault"
 _ "modernc.org/sqlite"
)
func TestVaultProfileConnectionAndRBAC(t *testing.T){
 root:=t.TempDir();if err:=os.Chmod(root,0700);err!=nil{t.Fatal(err)}
 key:=make([]byte,32);if _,err:=rand.Read(key);err!=nil{t.Fatal(err)}
 t.Setenv("VAULT_FILE",filepath.Join(root,"store.enc"))
 t.Setenv("VAULT_ACTIVE_KEY_B64",base64.StdEncoding.EncodeToString(key))
 v,err:=vault.FromEnv();if err!=nil{t.Fatal(err)}
 if err=v.Put("my_sqlite",vault.Credentials{Engine:"sqlite",Database:":memory:"});err!=nil{t.Fatal(err)}
 srv:=New("token")
 body:=[]byte(`{"name":"encrypted-profile","credentialRef":"my_sqlite"}`)
 req:=httptest.NewRequest("POST","/v1/connections",bytes.NewReader(body))
 req.Header.Set("Authorization","Bearer token")
 w:=httptest.NewRecorder();srv.ServeHTTP(w,req)
 if w.Code!=201||strings.Contains(w.Body.String(),"password"){t.Fatalf("vault connection: %d %s",w.Code,w.Body.String())}
 if !strings.Contains(w.Body.String(),`"readOnly":true`){t.Fatal("vault profile was not read-only")}
 req=httptest.NewRequest("POST","/v1/connections",bytes.NewReader(body))
 req.Header.Set("Authorization","Bearer token")
 scope,proof:=verifiedScope("token",strings.Repeat("a",64))
 req.Header.Set("X-ADB-Session-ID",scope)
 req.Header.Set("X-ADB-Session-Proof",proof)
 req.Header.Set("X-ADB-Role","viewer")
 req.Header.Set("X-ADB-Role-Proof",verifiedRole("token",scope,"viewer"))
 w=httptest.NewRecorder();srv.ServeHTTP(w,req)
 if w.Code!=403{t.Fatalf("viewer accessed vault: %d %s",w.Code,w.Body.String())}
	req=httptest.NewRequest("POST","/v1/connections",bytes.NewReader([]byte(`{"name":"bad","credentialRef":"my_sqlite","engine":"postgres"}`)))
 req.Header.Set("Authorization","Bearer token")
 w=httptest.NewRecorder();srv.ServeHTTP(w,req)
 if w.Code!=400{t.Fatalf("inline credential override accepted: %d",w.Code)}
}
