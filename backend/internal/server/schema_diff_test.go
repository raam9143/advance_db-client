package server
import("bytes";"net/http/httptest";"strings";"testing")
func TestDiffSchemasDeterministic(t *testing.T){
 before:=TableDescription{Name:"accounts",Columns:[]Column{{Name:"id",Type:"integer",PrimaryKey:true},{Name:"name",Type:"varchar(20)",Nullable:true}}}
 after:=TableDescription{Name:"accounts",Columns:[]Column{{Name:"id",Type:"bigint",PrimaryKey:true},{Name:"email",Type:"varchar(250)",Nullable:true}}}
 plan,err:=DiffSchemas(DiffInput{Before:before,After:after})
 if err!=nil{t.Fatal(err)}
 if !plan.RequiresReview||plan.Executable||len(plan.Changes)!=3{t.Fatalf("unexpected plan: %+v",plan)}
 if plan.Changes[0].Object!="email"||plan.Changes[2].Object!="name"{t.Fatalf("diff is not sorted: %+v",plan.Changes)}
}
func TestDiffSchemasInvalidInput(t *testing.T){
 q:=DiffInput{Before:TableDescription{Name:"a"},After:TableDescription{Name:"b"}}
 if _,err:=DiffSchemas(q);err==nil{t.Fatal("diffed unrelated tables")}
 q.After.Name="a";q.Before.Columns=[]Column{{Name:"x"},{Name:"x"}}
 if _,err:=DiffSchemas(q);err==nil{t.Fatal("accepted duplicate columns")}
}
func TestDiffEndpoint(t *testing.T){
 s:=New("secret")
 body:=`{"before":{"name":"u","columns":[]},"after":{"name":"u","columns":[]}}`
 r:=httptest.NewRequest("POST","/v1/schema/diff",bytes.NewBufferString(body));r.Header.Set("Authorization","Bearer secret")
 w:=httptest.NewRecorder();s.ServeHTTP(w,r)
 if w.Code!=200||!strings.Contains(w.Body.String(),`"executable":false`){t.Fatalf("%d %s",w.Code,w.Body.String())}
}
