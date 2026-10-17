package server

import(
 "net/http"
 "sync"
 "time"
)
type apiMetrics struct{
 mu sync.Mutex
 born time.Time
 total uint64
 byClass [5]uint64
 latency [5]uint64
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
