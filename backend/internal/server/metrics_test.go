package server
import(
 "encoding/json"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)
func TestAdminOnlyMetricsAndHealth(t *testing.T){
 s:=New("private-token")
 r:=httptest.NewRequest("GET","/v1/metrics",nil)
 w:=httptest.NewRecorder();s.ServeHTTP(w,r)
 if w.Code!=401{t.Fatalf("unauthenticated metrics: %d",w.Code)}
 r=httptest.NewRequest("GET","/v1/metrics",nil)
 r.Header.Set("Authorization","Bearer private-token")
 w=httptest.NewRecorder();s.ServeHTTP(w,r)
 if w.Code!=200{t.Fatal("admin metrics missing",w.Code)}
 var snapshot metricsSnapshot
 if e:=json.Unmarshal(w.Body.Bytes(),&snapshot);e!=nil||snapshot.Requests<1{t.Fatalf("%+v %v",snapshot,e)}
 raw:=w.Body.String()
 if strings.Contains(raw,"private-token")||strings.Contains(raw,"requestOwner"){t.Fatal("metrics leaked secrets or method names")}
 health:=httptest.NewRecorder();s.ServeHTTP(health,httptest.NewRequest("GET","/health",nil))
 if health.Code!=200{t.Fatal("health failed")}
 s.auditError=testingError{}
 health=httptest.NewRecorder();s.ServeHTTP(health,httptest.NewRequest("GET","/health",nil))
 if health.Code!=503{t.Fatal("audit-unhealthy server reported ready")}
}
type testingError struct{}
func(testingError)Error()string{return "error"}
func TestMetricsCountByStatusAndLatency(t *testing.T){
 m:=&apiMetrics{born:time.Now()}
 m.observe(200,30*time.Millisecond)
 m.observe(404,600*time.Millisecond)
 m.observe(503,6*time.Second)
 result:=m.snapshot()
 if result.Requests!=3||result.StatusClasses["2xx"]!=1||result.StatusClasses["4xx"]!=1||result.StatusClasses["5xx"]!=1||result.LatencyBuckets["gt5s"]!=1{t.Fatalf("%+v",result)}
}
