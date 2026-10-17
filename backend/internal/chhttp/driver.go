// Package chhttp offers a read-only HTTPS ClickHouse adapter for database/sql.
package chhttp
import("context";"crypto/tls";"database/sql";"database/sql/driver";"encoding/json";"errors";"fmt";"io";"net/http";"net/url";"regexp";"strconv";"strings";"time")
func init(){sql.Register("adbclickhouse",&Driver{})}
type Driver struct{}
func(*Driver)Open(dsn string)(driver.Conn,error){
 u,e:=url.Parse(dsn);if e!=nil||u==nil||u.Scheme!="https"||u.Hostname()==""||u.Port()==""||u.Path!="/"||u.User==nil||u.User.Username()==""||u.Fragment!=""{return nil,errors.New("ClickHouse requires verified HTTPS")}
 if v:=u.Query();len(v)!=1||v.Get("database")==""{return nil,errors.New("invalid ClickHouse database")}
 return &conn{u,&http.Client{Timeout:35*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse},Transport:&http.Transport{TLSClientConfig:&tls.Config{MinVersion:tls.VersionTLS12},Proxy:nil}}},nil
}
type conn struct{endpoint *url.URL;client *http.Client}
func(*conn)Close()error{return nil}
func(*conn)Prepare(string)(driver.Stmt,error){return nil,errors.New("prepared statements unsupported")}
func(*conn)Begin()(driver.Tx,error){return nil,errors.New("ClickHouse transactions unsupported")}
func(*conn)BeginTx(_ context.Context,o driver.TxOptions)(driver.Tx,error){if !o.ReadOnly{return nil,errors.New("read-only required")};return noTx{},nil}
type noTx struct{}
func(noTx)Commit()error{return nil}
func(noTx)Rollback()error{return nil}
func(c *conn)Ping(ctx context.Context)error{r,e:=c.QueryContext(ctx,"SELECT 1",nil);if e!=nil{return e};return r.Close()}
func(*conn)ExecContext(context.Context,string,[]driver.NamedValue)(driver.Result,error){return nil,errors.New("ClickHouse writes disabled")}
var paramName=regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
func readOnly(s string)bool{
 s=strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s),";"));if s==""||len(s)>100000||strings.Contains(s,";")||strings.Contains(s,"--")||strings.Contains(s,"/*")||strings.Contains(s,"*/"){return false}
 u:=" "+strings.ToUpper(s)+" "
 if !(strings.HasPrefix(u," SELECT ")||strings.HasPrefix(u," WITH ")||strings.HasPrefix(u," SHOW ")||strings.HasPrefix(u," DESCRIBE ")||strings.HasPrefix(u," EXPLAIN ")){return false}
 for _,k:=range []string{" INSERT "," UPDATE "," DELETE "," ALTER "," CREATE "," DROP "," TRUNCATE "," GRANT "," REVOKE "," FORMAT "," SETTINGS "," INTO OUTFILE"}{if strings.Contains(u,k){return false}}
 return true
}
func(c *conn)QueryContext(ctx context.Context,query string,args []driver.NamedValue)(driver.Rows,error){
 if !readOnly(query){return nil,errors.New("ClickHouse query must be read-only")}
 query=strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(query),";"))
 endpoint:=*c.endpoint;user:=endpoint.User.Username();password,_:=endpoint.User.Password();endpoint.User=nil
 v:=url.Values{"database":{endpoint.Query().Get("database")},"readonly":{"2"},"default_format":{"JSON"},"max_execution_time":{"30"},"max_result_rows":{"1001"},"result_overflow_mode":{"break"},"wait_end_of_query":{"1"}}
 for _,a:=range args{
  name:=a.Name;if name==""{name=fmt.Sprintf("p%d",a.Ordinal)};if !paramName.MatchString(name){return nil,errors.New("invalid parameter name")}
  var value string
  switch x:=a.Value.(type){case string:if len(x)>1<<20{return nil,errors.New("parameter too large")};value=x;case int64,float64,bool:value=fmt.Sprint(x);case nil:value="\\N";default:return nil,errors.New("invalid parameter type")}
  if v.Has("param_"+name){return nil,errors.New("duplicate parameter")};v.Set("param_"+name,value)
 }
 endpoint.RawQuery=v.Encode()
 req,e:=http.NewRequestWithContext(ctx,"POST",endpoint.String(),strings.NewReader(query));if e!=nil{return nil,e}
 req.SetBasicAuth(user,password);req.Header.Set("Content-Type","text/plain; charset=utf-8")
 res,e:=c.client.Do(req);if e!=nil{return nil,errors.New("ClickHouse HTTP request failed")};defer res.Body.Close()
 if res.StatusCode!=200{return nil,fmt.Errorf("ClickHouse returned HTTP %d",res.StatusCode)}
 b,e:=io.ReadAll(io.LimitReader(res.Body,(8<<20)+1));if e!=nil||len(b)>8<<20{return nil,errors.New("ClickHouse response too large")}
 var payload struct{Meta []struct{Name string `json:"name"`} `json:"meta"`;Data []map[string]any `json:"data"`;Exception string `json:"exception"`}
 dec:=json.NewDecoder(strings.NewReader(string(b)));dec.UseNumber()
 if dec.Decode(&payload)!=nil||payload.Exception!=""{return nil,errors.New("invalid ClickHouse result")}
 if len(payload.Meta)>500||len(payload.Data)>1001{return nil,errors.New("ClickHouse result limit exceeded")}
 cols:=make([]string,len(payload.Meta));rowsData:=make([][]driver.Value,len(payload.Data))
 for i,col:=range payload.Meta{cols[i]=col.Name}
 for i,row:=range payload.Data{
  rowsData[i]=make([]driver.Value,len(cols))
  for j,name:=range cols{
   switch x:=row[name].(type){
   case nil,string,float64,int64:rowsData[i][j]=x
   case json.Number:if n,e:=x.Int64();e==nil{rowsData[i][j]=n}else if f,e:=strconv.ParseFloat(string(x),64);e==nil{rowsData[i][j]=f}else{rowsData[i][j]=string(x)}
   case bool:if x{rowsData[i][j]=int64(1)}else{rowsData[i][j]=int64(0)}
   default:b,_:=json.Marshal(x);rowsData[i][j]=string(b)
   }
  }
 }
 return &rows{cols:cols,data:rowsData},nil
}
type rows struct{cols []string;data [][]driver.Value;index int}
func(r *rows)Columns()[]string{return r.cols}
func(r *rows)Close()error{return nil}
func(r *rows)Next(out []driver.Value)error{if r.index>=len(r.data){return io.EOF};copy(out,r.data[r.index]);r.index++;return nil}
