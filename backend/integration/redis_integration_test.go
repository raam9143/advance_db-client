package integration

import(
 "encoding/json"
 "os"
 "strings"
 "testing"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
func TestRedisLiveReadOnly(t *testing.T){
 if os.Getenv("TEST_REDIS_HOST")==""{t.Skip("requires TEST_REDIS_HOST")}
 t.Setenv("REDIS_ALLOW_PLAINTEXT_LOOPBACK","true")
 h:=server.New("integration-test")
 c:=server.ConnectionRequest{Name:"redis-ci",Engine:"redis",Host:os.Getenv("TEST_REDIS_HOST"),Port:6379,Database:"0",SSLMode:"disable"}
 status,data:=call(t,h,"POST","/v1/connections",c);check(t,status,data,201)
 var info server.PublicConnection
 if e:=json.Unmarshal(data,&info);e!=nil{t.Fatal(e)}
 if info.Engine!="redis"||!info.ReadOnly{t.Fatalf("unsafe connection: %+v",info)}
 path:="/v1/connections/"+info.ID
 status,data=call(t,h,"POST",path+"/redis",map[string]any{"command":"SCAN","cursor":"0","pattern":"*"})
 check(t,status,data,200)
 if !strings.Contains(string(data),`"command":"SCAN"`){t.Fatalf("bad scan result %s",data)}
 status,data=call(t,h,"POST",path+"/redis",map[string]any{"command":"SET","key":"key"})
 check(t,status,data,400)
 status,data=call(t,h,"POST",path+"/query",server.QueryRequest{SQL:"SELECT 1"})
 check(t,status,data,400)
 status,data=call(t,h,"GET",path+"/schema",nil);check(t,status,data,200)
 status,data=call(t,h,"DELETE",path,nil);check(t,status,data,204)
}
