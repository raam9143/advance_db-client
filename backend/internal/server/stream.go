package server

import(
 "context"
 "database/sql"
 "encoding/csv"
 "errors"
 "fmt"
 "net/http"
 "strconv"
 "time"
)
const (
 maxStreamRows=50000
 maxStreamBytes=32<<20
 maxStreamCols=250
)
type streamWriter struct{
 w http.ResponseWriter
 written int
}
func(w *streamWriter)Write(data []byte)(int,error){
 if len(data)>maxStreamBytes-w.written{return 0,errors.New("CSV export byte budget exceeded")}
 n,err:=w.w.Write(data);w.written+=n
 return n,err
}
func(s *Server)streamCSV(w http.ResponseWriter,r *http.Request,id string){
 c,found:=s.lookup(id,requestOwner(r))
 if !found{fail(w,404,"connection not found");return}
 if c.db==nil||!c.info.ReadOnly{fail(w,403,"streaming is allowed only on read-only SQL sessions");return}
 var req QueryRequest
 if !decode(w,r,&req){return}
 if !ReadOnlySQL(req.SQL){fail(w,400,"stream requires a single read-only SQL statement");return}
 args,err:=ValidateParameters(req.Parameters)
 if err!=nil{fail(w,400,err.Error());return}
 ctx,cancel:=context.WithTimeout(r.Context(),60*time.Second);defer cancel()
 var rows *sql.Rows
 var tx *sql.Tx
 // SQL Server and Oracle drivers do not reliably implement TxOptions.ReadOnly;
 // an actual database read-only login remains mandatory for these engines.
 if c.info.Engine=="oracle"||c.info.Engine=="sqlserver"{
  rows,err=c.db.QueryContext(ctx,req.SQL,args...)
 }else{
  tx,err=c.db.BeginTx(ctx,&sql.TxOptions{ReadOnly:true})
  if err==nil{rows,err=tx.QueryContext(ctx,req.SQL,args...)}
 }
 if tx!=nil{defer tx.Rollback()}
 if err!=nil{fail(w,400,"read-only stream query failed");return}
 defer rows.Close()
 cols,err:=rows.Columns()
 if err!=nil||len(cols)==0||len(cols)>maxStreamCols{fail(w,400,"stream column count out of bounds");return}
 w.Header().Set("Content-Type","text/csv; charset=utf-8")
 w.Header().Set("Content-Disposition","attachment; filename=\"query-stream.csv\"")
 w.Header().Set("X-ADB-Row-Cap",strconv.Itoa(maxStreamRows))
 w.Header().Set("X-ADB-Byte-Cap",strconv.Itoa(maxStreamBytes))
 w.Header().Set("Cache-Control","no-store")
 w.WriteHeader(http.StatusOK)
 budget:=&streamWriter{w:w}
 writer:=csv.NewWriter(budget)
 if err:=writer.Write(cols);err!=nil{return}
 writer.Flush();if writer.Error()!=nil{return}
 flush,hasFlush:=w.(http.Flusher)
 count:=0
 for rows.Next(){
  if ctx.Err()!=nil||count>=maxStreamRows{return}
  values:=make([]any,len(cols));ptr:=make([]any,len(cols))
  for i:=range values{ptr[i]=&values[i]}
  if err=rows.Scan(ptr...);err!=nil{return}
  record:=make([]string,len(cols))
  for i,value:=range values{
   switch b:=value.(type){case nil:record[i]="";case []byte:record[i]=safeCSVCell(string(b));default:record[i]=safeCSVCell(fmt.Sprint(value))}
   if len(record[i])>1<<20{return}
  }
  if err=writer.Write(record);err!=nil{return}
  writer.Flush()
  if writer.Error()!=nil{return}
  count++
  if hasFlush&&count%100==0{flush.Flush()}
 }
}
