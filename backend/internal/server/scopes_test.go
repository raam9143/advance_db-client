package server
import("net/http/httptest";"strings";"testing";"time")
func TestSessionProof(t *testing.T){
 s:=New("secret")
 id,proof:=verifiedScope("secret",strings.Repeat("a",64))
 req:=httptest.NewRequest("GET","/v1/connections",nil);req.Header.Set("Authorization","Bearer secret")
 req.Header.Set("X-ADB-Session-ID",id);req.Header.Set("X-ADB-Session-Proof",proof)
 w:=httptest.NewRecorder();s.ServeHTTP(w,req);if w.Code!=200{t.Fatalf("valid proof rejected: %s",w.Body.String())}
 req.Header.Set("X-ADB-Session-Proof",strings.Repeat("b",64))
 w=httptest.NewRecorder();s.ServeHTTP(w,req);if w.Code!=401{t.Fatal("invalid signature accepted")}
}
func TestConnectionOwnerIsolation(t *testing.T){
 s:=New("secret")
 id:=strings.Repeat("f",48)
 s.dbs[id]=&session{owner:"service",info:PublicConnection{ID:id},expiry:time.Now().Add(time.Hour)}
 r:=httptest.NewRequest("GET","/v1/connections",nil);r.Header.Set("Authorization","Bearer secret")
 w:=httptest.NewRecorder();s.ServeHTTP(w,r);if !strings.Contains(w.Body.String(),id){t.Fatal("own connection invisible")}
 scope,proof:=verifiedScope("secret",strings.Repeat("a",64))
 r.Header.Set("X-ADB-Session-ID",scope);r.Header.Set("X-ADB-Session-Proof",proof)
 w=httptest.NewRecorder();s.ServeHTTP(w,r);if strings.Contains(w.Body.String(),id){t.Fatal("cross-scope connection exposed")}
 r=httptest.NewRequest("DELETE","/v1/connections/"+id,nil)
 r.Header.Set("Authorization","Bearer secret");r.Header.Set("X-ADB-Session-ID",scope);r.Header.Set("X-ADB-Session-Proof",proof)
 w=httptest.NewRecorder();s.ServeHTTP(w,r);if w.Code!=404{t.Fatalf("cross-scope deletion got %d",w.Code)}
}
