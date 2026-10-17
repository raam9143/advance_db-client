package server
import(
 "bytes"
 "net/http/httptest"
 "strings"
 "testing"
)
func TestBrowserRoleHMACEnforcement(t *testing.T){
 token:="test-token"
 s:=New(token)
 id:=strings.Repeat("a",64)
 owner,proof:=verifiedScope(token,id)
 if owner!=id{t.Fatal("invalid scope")}
 for _,tc:=range []struct{role,roleProof string;status int}{
  {"admin",verifiedRole(token,id,"admin"),200},
  {"viewer",verifiedRole(token,id,"viewer"),200},
  {"admin",verifiedRole(token,id,"viewer"),401},
  {"root",verifiedRole(token,id,"root"),401},
  {"","",200},
 }{
  req:=httptest.NewRequest("GET","/v1/connections",nil)
  req.Header.Set("Authorization","Bearer "+token)
  req.Header.Set("X-ADB-Session-ID",id)
  req.Header.Set("X-ADB-Session-Proof",proof)
  if tc.role!=""{req.Header.Set("X-ADB-Role",tc.role);req.Header.Set("X-ADB-Role-Proof",tc.roleProof)}
  w:=httptest.NewRecorder();s.ServeHTTP(w,req)
  if w.Code!=tc.status{t.Fatalf("%s expected %d got %d: %s",tc.role,tc.status,w.Code,w.Body.String())}
 }
 viewer:=httptest.NewRequest("POST","/v1/ai/review",bytes.NewBufferString("{}"))
 viewer.Header.Set("Authorization","Bearer "+token)
 viewer.Header.Set("X-ADB-Session-ID",id)
 viewer.Header.Set("X-ADB-Session-Proof",proof)
 viewer.Header.Set("X-ADB-Role","viewer")
 viewer.Header.Set("X-ADB-Role-Proof",verifiedRole(token,id,"viewer"))
 w:=httptest.NewRecorder();s.ServeHTTP(w,viewer)
 if w.Code!=403{t.Fatalf("viewer AI review should be denied %d",w.Code)}
}
