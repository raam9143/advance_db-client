package server

import (
 "context"
 "database/sql"
 "errors"
 "fmt"
 "net/http"
 "net/url"
 "strconv"
 "strings"
 "time"
)
const maxPageSize=250
type Column struct {Name string `json:"name"`;Type string `json:"type"`;Nullable bool `json:"nullable"`;PrimaryKey bool `json:"primaryKey"`;Default *string `json:"default"`}
type Index struct {Name string `json:"name"`;Column string `json:"column"`;Unique bool `json:"unique"`}
type TableDescription struct {Schema string `json:"schema"`;Name string `json:"name"`;Columns []Column `json:"columns"`;Indexes []Index `json:"indexes"`}
type PageResult struct {QueryResult;Limit int `json:"limit"`;Offset int `json:"offset"`;NextOffset *int `json:"nextOffset"`}

func quoteIdentifier(engine,name string)(string,error){
 if strings.TrimSpace(name)==""||len(name)>128||strings.ContainsAny(name,"\x00\r\n"){return "",errors.New("invalid identifier")}
 if engine=="mysql"||engine=="mariadb"{return string(rune(96))+strings.ReplaceAll(name,string(rune(96)),string([]rune{96,96}))+string(rune(96)),nil}
 return string(rune(34))+strings.ReplaceAll(name,string(rune(34)),string([]rune{34,34}))+string(rune(34)),nil
}
func qualifiedName(engine,schema,table string)(string,error){
 name,err:=quoteIdentifier(engine,table);if err!=nil{return "",err}
 if engine=="sqlite"{if schema!=""&&schema!="main"{return "",errors.New("unsupported SQLite schema")};return name,nil}
 if schema==""{return "",errors.New("schema is required")}
 s,err:=quoteIdentifier(engine,schema);if err!=nil{return "",err};return s+"."+name,nil
}
func parsePage(q url.Values)(int,int,error){
 limit,offset:=100,0
 if raw:=q.Get("limit");raw!=""{v,e:=strconv.Atoi(raw);if e!=nil||v<1||v>maxPageSize{return 0,0,errors.New("limit must be between 1 and 250")};limit=v}
 if raw:=q.Get("offset");raw!=""{v,e:=strconv.Atoi(raw);if e!=nil||v<0||v>10000000{return 0,0,errors.New("offset out of range")};offset=v}
 return limit,offset,nil
}
func (s *Server) tablePage(w http.ResponseWriter,r *http.Request,id string){
 c,ok:=s.lookup(id);if !ok{fail(w,404,"connection not found");return}
 limit,offset,err:=parsePage(r.URL.Query());if err!=nil{fail(w,400,err.Error());return}
 name,err:=qualifiedName(c.info.Engine,r.URL.Query().Get("schema"),r.URL.Query().Get("table"));if err!=nil{fail(w,400,err.Error());return}
 query:=fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",name,limit+1,offset)
 ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
 tx,err:=c.db.BeginTx(ctx,&sql.TxOptions{ReadOnly:true});if err!=nil{fail(w,400,"could not open read-only transaction");return};defer tx.Rollback()
 rows,err:=tx.QueryContext(ctx,query);if err!=nil{fail(w,400,"could not read table");return};defer rows.Close()
 columns,err:=rows.Columns();if err!=nil{fail(w,500,"cannot read column names");return}
 out:=PageResult{QueryResult:QueryResult{Columns:columns,Rows:[][]any{}},Limit:limit,Offset:offset}
 for rows.Next(){
  if out.RowCount>=limit{next:=offset+limit;out.NextOffset=&next;out.Truncated=true;break}
  values:=make([]any,len(columns));ptr:=make([]any,len(columns));for i:=range values{ptr[i]=&values[i]}
  if err:=rows.Scan(ptr...);err!=nil{fail(w,500,"cannot scan table rows");return}
  for i,v:=range values{if b,ok:=v.([]byte);ok{values[i]=string(b)}}
  out.Rows=append(out.Rows,values);out.RowCount++
 }
 if rows.Err()!=nil{fail(w,500,"table read failed");return};respond(w,200,out)
}
func (s *Server) describe(w http.ResponseWriter,r *http.Request,id string){
 c,ok:=s.lookup(id);if !ok{fail(w,404,"connection not found");return}
 schema,table:=r.URL.Query().Get("schema"),r.URL.Query().Get("table")
 if _,err:=qualifiedName(c.info.Engine,schema,table);err!=nil{fail(w,400,err.Error());return}
 ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
 d:=TableDescription{Schema:schema,Name:table,Columns:[]Column{},Indexes:[]Index{}}
 var err error
 switch c.info.Engine{
 case "sqlite":err=describeSQLite(ctx,c.db,table,&d)
 case "postgres","cockroach":err=describeInformationSchema(ctx,c.db,"postgres",schema,table,&d)
 case "mysql","mariadb":err=describeInformationSchema(ctx,c.db,"mysql",schema,table,&d)
 default:err=errors.New("unsupported engine")
 }
 if err!=nil{fail(w,400,"could not inspect table");return};respond(w,200,d)
}
func describeSQLite(ctx context.Context,db *sql.DB,table string,d *TableDescription)error{
 q,err:=quoteIdentifier("sqlite",table);if err!=nil{return err}
 rows,err:=db.QueryContext(ctx,"PRAGMA table_info("+q+")");if err!=nil{return err}
 for rows.Next(){var cid,notNull,pk int;var name,typ string;var def sql.NullString
  if err:=rows.Scan(&cid,&name,&typ,&notNull,&def,&pk);err!=nil{rows.Close();return err}
  var v *string;if def.Valid{copy:=def.String;v=&copy}
  d.Columns=append(d.Columns,Column{Name:name,Type:typ,Nullable:notNull==0,PrimaryKey:pk>0,Default:v})
 }
 err=rows.Err();rows.Close();if err!=nil{return err}
 idx,err:=db.QueryContext(ctx,"PRAGMA index_list("+q+")");if err!=nil{return err}
 type info struct{name string;unique bool};indexes:=[]info{}
 for idx.Next(){var seq,unique,partial int;var name,origin string
  if err:=idx.Scan(&seq,&name,&unique,&origin,&partial);err!=nil{idx.Close();return err}
  indexes=append(indexes,info{name,unique!=0})
 }
 err=idx.Err();idx.Close();if err!=nil{return err}
 for _,index:=range indexes{
  name,err:=quoteIdentifier("sqlite",index.name);if err!=nil{return err}
  cols,err:=db.QueryContext(ctx,"PRAGMA index_info("+name+")");if err!=nil{return err}
  for cols.Next(){var seq,cid int;var column sql.NullString
   if err:=cols.Scan(&seq,&cid,&column);err!=nil{cols.Close();return err}
   if column.Valid{d.Indexes=append(d.Indexes,Index{Name:index.name,Column:column.String,Unique:index.unique})}
  }
  err=cols.Err();cols.Close();if err!=nil{return err}
 }
 return nil
}
func describeInformationSchema(ctx context.Context,db *sql.DB,kind,schema,table string,d *TableDescription)error{
 placeholder:=func(n int)string{if kind=="postgres"{return fmt.Sprintf("$%d",n)};return "?"}
 q:="SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema="+placeholder(1)+" AND table_name="+placeholder(2)+" ORDER BY ordinal_position"
 rows,err:=db.QueryContext(ctx,q,schema,table);if err!=nil{return err}
 for rows.Next(){var name,typ,nullable string;var def sql.NullString
  if err:=rows.Scan(&name,&typ,&nullable,&def);err!=nil{rows.Close();return err}
  var v *string;if def.Valid{copy:=def.String;v=&copy}
  d.Columns=append(d.Columns,Column{Name:name,Type:typ,Nullable:nullable=="YES",Default:v})
 }
 err=rows.Err();rows.Close();if err!=nil{return err}
 if kind=="postgres"{
  q="SELECT a.attname FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_namespace n ON n.oid=t.relnamespace JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum = ANY(i.indkey) WHERE i.indisprimary AND n.nspname=$1 AND t.relname=$2"
 }else{
  q="SELECT column_name FROM information_schema.key_column_usage WHERE table_schema=? AND table_name=? AND constraint_name='PRIMARY'"
 }
 primary,err:=db.QueryContext(ctx,q,schema,table);if err!=nil{return err}
 keys:=map[string]bool{};for primary.Next(){var key string;if err:=primary.Scan(&key);err!=nil{primary.Close();return err};keys[key]=true}
 err=primary.Err();primary.Close();if err!=nil{return err}
 for i:=range d.Columns{d.Columns[i].PrimaryKey=keys[d.Columns[i].Name]}
 if kind=="postgres"{
  q="SELECT ix.relname, a.attname, i.indisunique FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace JOIN pg_index i ON i.indrelid=t.oid JOIN pg_class ix ON ix.oid=i.indexrelid JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum = ANY(i.indkey) WHERE n.nspname=$1 AND t.relname=$2"
 }else{
  q="SELECT index_name,column_name,non_unique FROM information_schema.statistics WHERE table_schema=? AND table_name=? ORDER BY index_name,seq_in_index"
 }
 ix,err:=db.QueryContext(ctx,q,schema,table);if err!=nil{return err}
 for ix.Next(){var name,col string;var unique bool
  if kind=="postgres"{if err=ix.Scan(&name,&col,&unique);err!=nil{ix.Close();return err}}
  if kind=="mysql"{var notUnique int;if err=ix.Scan(&name,&col,&notUnique);err!=nil{ix.Close();return err};unique=notUnique==0}
  d.Indexes=append(d.Indexes,Index{Name:name,Column:col,Unique:unique})
 }
 err=ix.Err();ix.Close();return err
}
