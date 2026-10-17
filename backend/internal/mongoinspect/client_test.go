package mongoinspect

import (
 "encoding/json"
 "strings"
 "testing"
)
func TestValidateMongoEndpoint(t *testing.T){
 for _,c:=range []Config{
  {Host:"example.com",Port:27017,Database:"app",PlaintextLoopback:true},
  {Host:"localhost",Port:27017,Database:"db bad"},
  {Host:"localhost",Port:0,Database:"db"},
  {Host:"localhost",Port:27017,Database:"app",Password:"x"},
  {Host:"localhost\nattack",Port:27017,Database:"app"},
 }{
  if err:=ValidateConfig(c);err==nil{t.Fatalf("accepted %+v",c)}
 }
 if err:=ValidateConfig(Config{Host:"127.0.0.1",Port:27017,Database:"app",PlaintextLoopback:true});err!=nil{t.Fatal(err)}
}
func TestReadOnlyFindInput(t *testing.T){
 good:=FindRequest{Collection:"users",Limit:50,Offset:0,FilterField:"status",FilterValue:json.RawMessage(`"active"`)}
 filter,err:=ValidateFind(good)
 if err!=nil||len(filter)!=1{t.Fatalf("good filter: %+v %v",filter,err)}
 for _,c:=range []FindRequest{
  {Collection:"system.users",Limit:10},
  {Collection:"$cmd",Limit:10},
  {Collection:"orders",Limit:101},
  {Collection:"orders",Limit:50,Offset:10001},
  {Collection:"orders",Limit:20,FilterField:"$where",FilterValue:json.RawMessage(`"javascript"`)},
  {Collection:"orders",Limit:20,FilterField:"a..b",FilterValue:json.RawMessage(`"x"`)},
  {Collection:"orders",Limit:20,FilterField:"x",FilterValue:json.RawMessage(`{"$gt":0}`)},
  {Collection:"orders",Limit:20,FilterField:"x",FilterValue:json.RawMessage(`[1,2]`)},
  {Collection:"orders",Limit:20,FilterField:"x",FilterValue:json.RawMessage(strings.Repeat("x",5000))},
 }{
  if _,err=ValidateFind(c);err==nil{t.Fatalf("accepted unsafe request %+v",c)}
 }
}
