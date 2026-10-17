package server
import(
 "context"
 "encoding/json"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)
func TestAIReviewPrivacy(t *testing.T){
 for _,req:=range []AIReviewRequest{
 {SQL:"SELECT * FROM t",Purpose:"explain",Consent:false},
 {SQL:"DELETE FROM t",Purpose:"explain",Consent:true},
 {SQL:"SELECT secret_key FROM t",Purpose:"explain",Consent:true},
 {SQL:"SELECT * FROM t",Purpose:"delete",Consent:true},
 }{if _,err:=sanitizeAI(req);err==nil{t.Fatalf("unsafe AI input accepted %+v",req)}}
 sqlText,err:=sanitizeAI(AIReviewRequest{SQL:"SELECT * FROM accounts WHERE email='person@example.com'",Purpose:"optimize",Consent:true})
 if err!=nil{t.Fatal(err)}
 if strings.Contains(sqlText,"person@example.com")||!strings.Contains(sqlText,"REDACTED"){t.Fatalf("SQL not redacted: %s",sqlText)}
}
func TestAIProviderMockAndNoToolExecution(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  auth:=r.Header.Get("Authorization");if auth!="Bearer fake-key"{t.Error("missing server-only token")}
  data,err:=io.ReadAll(r.Body);if err!=nil{t.Fatal(err)}
  var payload map[string]any;_ =json.Unmarshal(data,&payload)
  if payload["store"]!=false{t.Error("AI request retention not disabled")}
  if _,ok:=payload["tools"];!ok{t.Error("tools were not explicitly disabled")}
  _,_=w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"Use a selective index after benchmarking."}]}]}`))
 }))
 defer server.Close()
 result,err:=callAI(context.Background(),server.Client(),server.URL,"fake-key","gpt-4o-mini","optimize","SELECT 1")
 if err!=nil||result.Executed||!strings.Contains(result.Advice,"index"){t.Fatalf("%+v %v",result,err)}
}
func TestAIEndpointDisabledByDefault(t *testing.T){
 t.Setenv("AI_ASSIST_ENABLED","false")
 s:=New("secret")
 req:=httptest.NewRequest("POST","/v1/ai/review",strings.NewReader(`{"sql":"SELECT 1","purpose":"explain","consent":true}`))
 req.Header.Set("Authorization","Bearer secret")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=503{t.Fatalf("disabled AI endpoint returned %d",w.Code)}
}
