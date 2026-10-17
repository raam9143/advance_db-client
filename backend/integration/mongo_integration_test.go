package integration

import (
 "context"
 "encoding/json"
 "os"
 "strings"
 "testing"
 "time"

 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
func TestMongoLiveReadonly(t *testing.T){
 host:=os.Getenv("TEST_MONGO_HOST")
 if host==""{t.Skip("requires TEST_MONGO_HOST")}
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
 // Test setup alone seeds the disposable database; app exposes NO MongoDB write path.
 native,err:=mongo.Connect(ctx,options.Client().SetHosts([]string{host+":27017"}).SetDirect(true))
 if err!=nil{t.Fatal(err)}
 defer native.Disconnect(context.Background())
 db:=native.Database("adb_mongo_ci")
 coll:=db.Collection("adb_documents")
 defer coll.Drop(context.Background())
 if _,err=coll.InsertMany(ctx,[]interface{}{bson.M{"label":"first","seq":1},bson.M{"label":"second","seq":2},bson.M{"label":"third","seq":3}});err!=nil{t.Fatal(err)}
 t.Setenv("MONGO_ALLOW_PLAINTEXT_LOOPBACK","true")
 h:=server.New("integration-test")
 c:=server.ConnectionRequest{Name:"mongodb ci",Engine:"mongo",Host:host,Port:27017,Database:"adb_mongo_ci",SSLMode:"disable",ReadOnly:true}
 status,body:=call(t,h,"POST","/v1/connections",c);check(t,status,body,201)
 var info server.PublicConnection
 if e:=json.Unmarshal(body,&info);e!=nil{t.Fatal(e)}
 if info.Engine!="mongo"||!info.ReadOnly{t.Fatalf("unexpected MongoDB session: %+v",info)}
 base:="/v1/connections/"+info.ID
 status,body=call(t,h,"GET",base+"/schema",nil);check(t,status,body,200)
 if !strings.Contains(string(body),"adb_documents"){t.Fatalf("collection missing: %s",body)}
 status,body=call(t,h,"POST",base+"/mongo/find",map[string]any{"collection":"adb_documents","limit":2,"offset":0})
 check(t,status,body,200)
 var first struct{Documents []json.RawMessage `json:"documents"`;NextOffset *int `json:"nextOffset"`}
 if err:=json.Unmarshal(body,&first);err!=nil||len(first.Documents)!=2||first.NextOffset==nil||*first.NextOffset!=2{t.Fatalf("pagination failed: %v %s",err,body)}
 status,body=call(t,h,"POST",base+"/mongo/find",map[string]any{"collection":"adb_documents","limit":20,"offset":0,"filterField":"label","filterValue":"second"})
 check(t,status,body,200)
 if !strings.Contains(string(body),"second")||strings.Contains(string(body),"first"){t.Fatalf("equality match failed: %s",body)}
 status,body=call(t,h,"POST",base+"/mongo/find",map[string]any{"collection":"adb_documents","limit":10,"offset":0,"filterField":"label","filterValue":map[string]any{"$ne":"first"}})
 check(t,status,body,400)
 for _,endpoint:=range []string{"/query","/explain","/export","/redis"}{
  status,body=call(t,h,"POST",base+endpoint,server.QueryRequest{SQL:"DELETE FROM adb_documents"})
  check(t,status,body,400)
 }
 status,body=call(t,h,"DELETE",base,nil);check(t,status,body,204)
}
