package server

import("strings";"testing")
func TestCapabilities(t *testing.T){
 got:=Capabilities()
 if len(got)!=6{t.Fatalf("wanted 6, got %d",len(got))}
 for _,v:=range got{if !v.Schema||(v.Engine!="clickhouse"&&!v.Explain)||!v.Parameters||!v.CSVExport||!v.JSONExport{t.Fatalf("incomplete: %+v",v)}}
}
func TestValidateParameters(t *testing.T){
 v,err:=ValidateParameters([]any{"safe",float64(3),true,nil})
 if err!=nil||len(v)!=4{t.Fatalf("%v %+v",err,v)}
 if _,err=ValidateParameters([]any{map[string]any{"x":1}});err==nil{t.Fatal("object accepted")}
 if _,err=ValidateParameters([]any{strings.Repeat("X",(1<<20)+1)});err==nil{t.Fatal("oversized value accepted")}
 if _,err=ValidateParameters(make([]any,101));err==nil{t.Fatal("too many parameters")}
}
func TestExplainGuard(t *testing.T){
 for _,q:=range []string{"DROP TABLE u","SELECT 1; DELETE FROM u","EXPLAIN SELECT 1","SELECT * FROM u FOR UPDATE"}{
  if _,err:=explainStatement("postgres",q);err==nil{t.Fatalf("unsafe: %s",q)}
 }
 for _,engine:=range []string{"postgres","cockroach","mysql","mariadb","sqlite"}{
  stmt,err:=explainStatement(engine,"SELECT 1;")
  if err!=nil||!strings.Contains(stmt,"SELECT 1"){t.Fatalf("%s: %s %v",engine,stmt,err)}
 }
}
