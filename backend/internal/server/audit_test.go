package server
import(
 "bytes"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "testing"
 "net/http/httptest"
)
func TestAuditChainedRedacted(t *testing.T){
 root:=t.TempDir()
 if e:=os.Chmod(root,0700);e!=nil{t.Fatal(e)}
 file:=filepath.Join(root,"audit.jsonl")
 t.Setenv("AUDIT_LOG_PATH",file)
 t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("x",40))
 a,err:=openAuditLog();if err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup
 for i:=0;i<12;i++{
  wg.Add(1);go func(){defer wg.Done();if err:=a.Append("POST","/v1/query/sql/SECRET","user123","started",0);err!=nil{t.Error(err)}}()
 }
 wg.Wait();_ =a.file.Close()
 raw,err:=os.ReadFile(file);if err!=nil{t.Fatal(err)}
 if bytes.Contains(raw,[]byte("SECRET"))||bytes.Contains(raw,[]byte("user123")){t.Fatal("audit leaked sensitive identifiers")}
 reopened,err:=openAuditLog();if err!=nil{t.Fatal(err)}
 _=reopened.file.Close()
 if err=os.WriteFile(file,bytes.Replace(raw,[]byte("started"),[]byte("stopped"),1),0600);err!=nil{t.Fatal(err)}
 if opened,err:=openAuditLog();err==nil{_=opened.file.Close();t.Fatal("tamper accepted")}
}
func TestAuditFailClosedAndRecords(t *testing.T){
 root:=t.TempDir()
 _=os.Chmod(root,0700)
 t.Setenv("AUDIT_LOG_PATH",filepath.Join(root,"audit.jsonl"))
 t.Setenv("AUDIT_HMAC_KEY","weak")
 s:=New("token")
 req:=httptest.NewRequest("GET","/v1/connections",nil);req.Header.Set("Authorization","Bearer token")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=503{t.Fatalf("invalid audit configuration accepted: %d",w.Code)}
 t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("y",40))
 s=New("token")
 req=httptest.NewRequest("GET","/v1/connections",nil);req.Header.Set("Authorization","Bearer token")
 w=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=200{t.Fatalf("valid audit rejected: %d %s",w.Code,w.Body.String())}
 raw,err:=os.ReadFile(filepath.Join(root,"audit.jsonl"));if err!=nil{t.Fatal(err)}
 if !bytes.Contains(raw,[]byte("finished"))||!bytes.Contains(raw,[]byte("started")){t.Fatalf("missing audit events: %s",raw)}
}

func TestAuditIsSingleWriterAndFailsReadiness(t *testing.T){
 root:=t.TempDir();if err:=os.Chmod(root,0700);err!=nil{t.Fatal(err)}
 path:=filepath.Join(root,"api.jsonl")
 t.Setenv("AUDIT_LOG_PATH",path)
 t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("z",40))
 s:=New("token")
 if s.audit==nil||s.auditError!=nil{t.Fatalf("could not open audit: %v",s.auditError)}
 if other,err:=openAuditLog();err==nil{
  _=other.file.Close()
  t.Fatal("two audit writers acquired same file")
 }
 if err:=s.audit.file.Close();err!=nil{t.Fatal(err)}
 req:=httptest.NewRequest("GET","/v1/connections",nil)
 req.Header.Set("Authorization","Bearer token")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=503{t.Fatalf("audit error did not fail closed: %d",w.Code)}
 req=httptest.NewRequest("GET","/health",nil)
 w=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=503{t.Fatalf("health was ready after audit failure: %d",w.Code)}
}
