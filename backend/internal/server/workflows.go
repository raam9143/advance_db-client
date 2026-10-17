package server

import (
"errors"
"fmt"
"net/http"
"strings"
)
type EngineCapability struct {
 Engine string `json:"engine"`
 Protocol string `json:"protocol"`
 Schema bool `json:"schema"`
 Explain bool `json:"explain"`
 Parameters bool `json:"parameters"`
 CSVExport bool `json:"csvExport"`
 JSONExport bool `json:"jsonExport"`
}
func Capabilities() []EngineCapability {
 return []EngineCapability{
 {Engine:"postgres",Protocol:"postgresql",Schema:true,Explain:true,Parameters:true,CSVExport:true,JSONExport:true},
 {Engine:"cockroach",Protocol:"postgresql",Schema:true,Explain:true,Parameters:true,CSVExport:true,JSONExport:true},
 {Engine:"mysql",Protocol:"mysql",Schema:true,Explain:true,Parameters:true,CSVExport:true,JSONExport:true},
 {Engine:"mariadb",Protocol:"mysql",Schema:true,Explain:true,Parameters:true,CSVExport:true,JSONExport:true},
 {Engine:"sqlite",Protocol:"sqlite",Schema:true,Explain:true,Parameters:true,CSVExport:true,JSONExport:true},
 }
}
func ValidateParameters(params []any) ([]any,error) {
 if len(params)>100{return nil,errors.New("maximum 100 query parameters")}
 out:=make([]any,len(params))
 for i,p:=range params {
  switch v:=p.(type) {
  case nil,bool:out[i]=v
  case float64:out[i]=v
  case int,int64:out[i]=v
  case string:
   if len(v)>1<<20{return nil,fmt.Errorf("parameter %d exceeds 1 MB",i+1)}
   out[i]=v
  default:return nil,fmt.Errorf("unsupported type for parameter %d",i+1)
  }
 }
 return out,nil
}
func explainStatement(engine,query string)(string,error) {
 if !ReadOnlySQL(query){return "",errors.New("EXPLAIN requires a single read-only query")}
 trimmed:=strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(query),";"))
 if strings.HasPrefix(strings.ToUpper(trimmed),"EXPLAIN "){return "",errors.New("EXPLAIN input must be SELECT or WITH")}
 upper:=strings.ToUpper(trimmed)
 if !(strings.HasPrefix(upper,"SELECT ")||strings.HasPrefix(upper,"WITH ")){return "",errors.New("EXPLAIN input must be SELECT or WITH")}
 switch engine {
 case "sqlite":return "EXPLAIN QUERY PLAN "+trimmed,nil
 case "postgres","cockroach","mysql","mariadb":return "EXPLAIN "+trimmed,nil
 default:return "",errors.New("engine does not support EXPLAIN")
 }
}
func (s *Server) explain(w http.ResponseWriter,r *http.Request,id string) {
 c,ok:=s.lookup(id)
 if !ok{fail(w,http.StatusNotFound,"connection not found");return}
 var q QueryRequest
 if !decode(w,r,&q){return}
 stmt,err:=explainStatement(c.info.Engine,q.SQL)
 if err!=nil{fail(w,http.StatusBadRequest,err.Error());return}
 q.SQL=stmt
 q.ConfirmWrite=false
 if q.TimeoutSeconds<=0{q.TimeoutSeconds=15}
 out,err:=s.run(r.Context(),c,q)
 if err!=nil{fail(w,http.StatusBadRequest,err.Error());return}
 respond(w,200,out)
}
