package server
import(
 "net/http/httptest"
 "strings"
 "testing"
)
func TestPrometheusMetricsAreRedactedAndRoleBound(t *testing.T){
 s:=New("token")
 for i:=0;i<4;i++{s.metrics.observe(500,10)}
 req:=httptest.NewRequest("GET","/v1/metrics/prometheus",nil)
 req.Header.Set("Authorization","Bearer token")
 w:=httptest.NewRecorder();s.ServeHTTP(w,req)
 body:=w.Body.String()
 if w.Code!=200||!strings.Contains(body,`adb_http_responses_total{status_class="5xx"} 4`)||strings.Contains(body,"token"){t.Fatalf("bad scrape %d %s",w.Code,body)}
 scope,proof:=verifiedScope("token",strings.Repeat("b",64))
 req=httptest.NewRequest("GET","/v1/metrics/prometheus",nil)
 req.Header.Set("Authorization","Bearer token")
 req.Header.Set("X-ADB-Session-ID",scope)
 req.Header.Set("X-ADB-Session-Proof",proof)
 req.Header.Set("X-ADB-Role","viewer")
 req.Header.Set("X-ADB-Role-Proof",verifiedRole("token",scope,"viewer"))
 w=httptest.NewRecorder();s.ServeHTTP(w,req)
 if w.Code!=403{t.Fatalf("viewer scraped metrics: %d",w.Code)}
}
