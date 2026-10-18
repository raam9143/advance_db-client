package server
import(
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "testing"
)
func TestAuditHTTPSinkAndChain(t *testing.T){
 var mu sync.Mutex
 events:=[]auditEntry{}
 server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Authorization")!="Bearer "+strings.Repeat("s",40){t.Error("bad sink token")}
  var entry auditEntry
  if json.NewDecoder(r.Body).Decode(&entry)!=nil{t.Error("invalid event")}
  mu.Lock();events=append(events,entry);mu.Unlock()
  w.WriteHeader(204)
 }))
 defer server.Close()
 root:=t.TempDir();_ =os.Chmod(root,0700)
 t.Setenv("AUDIT_LOG_PATH",filepath.Join(root,"audit.jsonl"))
 t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("k",40))
 t.Setenv("AUDIT_SINK_URL",server.URL)
 t.Setenv("AUDIT_SINK_TOKEN",strings.Repeat("s",40))
 a,err:=openAuditLog();if err!=nil{t.Fatal(err)}
 a.sinkClient=server.Client()
 if err=a.Append("POST","/v1/connections","owner-one","started",0);err!=nil{t.Fatal(err)}
 if err=a.Append("POST","/v1/connections","owner-one","finished",201);err!=nil{t.Fatal(err)}
 mu.Lock();defer mu.Unlock()
 if len(events)!=2||events[1].Previous!=events[0].MAC||events[0].MAC==""{t.Fatalf("bad audit chain %+v",events)}
 if events[0].PathHash=="/v1/connections"||events[0].OwnerHash=="owner-one"{t.Fatal("audit identifiers leaked")}
 _=a.file.Close()
}
func TestAuditSinkDenialAndInsecureURL(t *testing.T){
 root:=t.TempDir();_ =os.Chmod(root,0700)
 t.Setenv("AUDIT_LOG_PATH",filepath.Join(root,"events.jsonl"))
 t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("k",40))
 t.Setenv("AUDIT_SINK_TOKEN",strings.Repeat("s",40))
 t.Setenv("AUDIT_SINK_URL","http://insecure.example.org/ingest")
 if a,err:=openAuditLog();err==nil{_ =a.file.Close();t.Fatal("plaintext audit sink accepted")}
 t.Setenv("AUDIT_SINK_URL","")
 a,err:=openAuditLog();if err!=nil{t.Fatal(err)}
 a.sinkURL="https://sink.example.org/ingest"
 a.sinkClient=&http.Client{Transport:roundTripReject{}}
 if err=a.Append("GET","/v1/test","owner","started",0);err==nil{t.Fatal("failed sink accepted")}
 _=a.file.Close()
}
type roundTripReject struct{}
func(roundTripReject)RoundTrip(*http.Request)(*http.Response,error){return &http.Response{StatusCode:503,Body:http.NoBody,Header:http.Header{}},nil}
