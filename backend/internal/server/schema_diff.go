package server
import(
 "errors"
 "net/http"
 "sort"
 "strings"
)
// DiffInput compares table metadata snapshots only. No DDL is generated or run.
type DiffInput struct{
 Before TableDescription `json:"before"`
 After TableDescription `json:"after"`
}
type DiffChange struct{
 Kind string `json:"kind"`
 Object string `json:"object"`
 Previous string `json:"previous,omitempty"`
 Proposed string `json:"proposed,omitempty"`
 Destructive bool `json:"destructive"`
}
type DiffPlan struct{
 Table string `json:"table"`
 Changes []DiffChange `json:"changes"`
 RequiresReview bool `json:"requiresReview"`
 Executable bool `json:"executable"`
}
func checkDefinition(d TableDescription)error{
 if strings.TrimSpace(d.Name)==""||len(d.Name)>128||len(d.Schema)>128{return errors.New("invalid table identity")}
 for _,x:=range []string{d.Name,d.Schema}{if strings.ContainsAny(x,"\x00\r\n"){return errors.New("invalid table identity")}}
 if len(d.Columns)>2048||len(d.Indexes)>2048{return errors.New("schema exceeds maximum size")}
 seen:=map[string]bool{}
 for _,c:=range d.Columns{
  if strings.TrimSpace(c.Name)==""||len(c.Name)>128||seen[c.Name]{return errors.New("duplicate or invalid column name")}
  seen[c.Name]=true
  if len(c.Type)>256{return errors.New("column type too long")}
 }
 return nil
}
func DiffSchemas(input DiffInput)(DiffPlan,error){
 if err:=checkDefinition(input.Before);err!=nil{return DiffPlan{},err}
 if err:=checkDefinition(input.After);err!=nil{return DiffPlan{},err}
 if input.Before.Name!=input.After.Name{return DiffPlan{},errors.New("table names must match")}
 result:=DiffPlan{Table:input.Before.Name,Changes:[]DiffChange{},Executable:false}
 old:=map[string]Column{}
 newCols:=map[string]Column{}
 for _,c:=range input.Before.Columns{old[c.Name]=c}
 for _,c:=range input.After.Columns{newCols[c.Name]=c}
 for name,before:=range old{
  after,exists:=newCols[name]
  if !exists{result.Changes=append(result.Changes,DiffChange{Kind:"drop_column",Object:name,Previous:before.Type,Destructive:true});continue}
  if !strings.EqualFold(before.Type,after.Type){result.Changes=append(result.Changes,DiffChange{Kind:"change_type",Object:name,Previous:before.Type,Proposed:after.Type,Destructive:true})}
  if before.Nullable!=after.Nullable{result.Changes=append(result.Changes,DiffChange{Kind:"change_nullable",Object:name,Previous:boolDescription(before.Nullable),Proposed:boolDescription(after.Nullable),Destructive:!after.Nullable})}
  if before.PrimaryKey!=after.PrimaryKey{result.Changes=append(result.Changes,DiffChange{Kind:"change_primary_key",Object:name,Previous:boolDescription(before.PrimaryKey),Proposed:boolDescription(after.PrimaryKey),Destructive:true})}
 }
 for name,after:=range newCols{
  if _,ok:=old[name];!ok{result.Changes=append(result.Changes,DiffChange{Kind:"add_column",Object:name,Proposed:after.Type,Destructive:!after.Nullable&&after.Default==nil})}
 }
 sort.Slice(result.Changes,func(i,j int)bool{
  if result.Changes[i].Object==result.Changes[j].Object{return result.Changes[i].Kind<result.Changes[j].Kind}
  return result.Changes[i].Object<result.Changes[j].Object
 })
 result.RequiresReview=len(result.Changes)>0
 return result,nil
}
func boolDescription(v bool)string{if v{return "true"};return "false"}
func (s *Server) diffSchema(w http.ResponseWriter,r *http.Request){
 var q DiffInput
 if !decode(w,r,&q){return}
 result,err:=DiffSchemas(q)
 if err!=nil{fail(w,http.StatusBadRequest,err.Error());return}
 respond(w,200,result)
}
