package server

import(
 "net/http"
 "fmt"
 "strings"
 "sync"
 "time"
)
type apiMetrics struct{
 mu sync.Mutex
 born time.Time
 total uint64
 byClass [5]uint64
 latency [5]uint64
 durationSum float64
}
type metricsResponseWriter struct{http.ResponseWriter;status int}
func(w *metricsResponseWriter)WriteHeader(code int){w.status=code;w.ResponseWriter.WriteHeader(code)}
func(w *metricsResponseWriter)Write(b []byte)(int,error){return w.ResponseWriter.Write(b)}
type metricsSnapshot struct{
 UptimeSeconds int64 `json:"uptimeSeconds"`
 Requests uint64 `json:"requestsTotal"`
 StatusClasses map[string]uint64 `json:"statusClasses"`
 LatencyBuckets map[string]uint64 `json:"latencyBuckets"`
}
func(m *apiMetrics)observe(status int,elapsed time.Duration){
 m.mu.Lock();defer m.mu.Unlock()
 m.total++
 m.durationSum+=elapsed.Seconds()
 n:=status/100-1;if n<0||n>4{n=4}
 m.byClass[n]++
 switch{
 case elapsed<=50*time.Millisecond:m.latency[0]++
 case elapsed<=200*time.Millisecond:m.latency[1]++
 case elapsed<=time.Second:m.latency[2]++
 case elapsed<=5*time.Second:m.latency[3]++
 default:m.latency[4]++
 }
}
func(m *apiMetrics)snapshot()metricsSnapshot{
 m.mu.Lock();defer m.mu.Unlock()
 return metricsSnapshot{UptimeSeconds:int64(time.Since(m.born).Seconds()),Requests:m.total,
  StatusClasses:map[string]uint64{"1xx":m.byClass[0],"2xx":m.byClass[1],"3xx":m.byClass[2],"4xx":m.byClass[3],"5xx":m.byClass[4]},
  LatencyBuckets:map[string]uint64{"le50ms":m.latency[0],"le200ms":m.latency[1],"le1s":m.latency[2],"le5s":m.latency[3],"gt5s":m.latency[4]}}
}

func(m *apiMetrics)prometheus()string{
 snap:=m.snapshot()
 var b strings.Builder
 fmt.Fprintf(&b,"# HELP adb_http_requests_total Total API requests\n# TYPE adb_http_requests_total counter\nadb_http_requests_total %d\n",snap.Requests)
 b.WriteString("# HELP adb_http_responses_total API responses by status class\n# TYPE adb_http_responses_total counter\n")
 for _,class:=range []string{"1xx","2xx","3xx","4xx","5xx"}{
  fmt.Fprintf(&b,"adb_http_responses_total{status_class=%q} %d\n",class,snap.StatusClasses[class])
 }
 // Buckets are cumulative and have fixed labels to avoid high cardinality.
 m.mu.Lock();v:=m.latency;seconds:=m.durationSum;m.mu.Unlock()
 b.WriteString("# HELP adb_http_request_duration_seconds Request duration histogram\n# TYPE adb_http_request_duration_seconds histogram\n")
 sum:=uint64(0)
 for i,threshold:=range []string{"0.05","0.2","1","5","+Inf"}{
  sum+=v[i]
  fmt.Fprintf(&b,"adb_http_request_duration_seconds_bucket{le=%q} %d\n",threshold,sum)
 }
 fmt.Fprintf(&b,"adb_http_request_duration_seconds_count %d\n",snap.Requests)
 fmt.Fprintf(&b,"adb_http_request_duration_seconds_sum %.6f\n",seconds)
 fmt.Fprintf(&b,"# HELP adb_uptime_seconds Process uptime\n# TYPE adb_uptime_seconds gauge\nadb_uptime_seconds %d\n",snap.UptimeSeconds)
 return b.String()
}
