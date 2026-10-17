// Command adbctl is a conservative read-only CLI for Advance DB Client API.
package main
import(
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "regexp"
 "strings"
 "time"
)
var connectionID=regexp.MustCompile(`^[a-f0-9]{48}$`)
func call(ctx context.Context,method,path string,data any,out io.Writer,base,token string)error{
 if token==""{return errors.New("ADB_API_TOKEN is required")}
 u,err:=url.Parse(base)
 if err!=nil||u==nil||!(u.Scheme=="http"||u.Scheme=="https")||u.Host==""||u.User!=nil||u.RawQuery!=""||u.Fragment!=""||u.Path!=""&&u.Path!="/"{return errors.New("invalid ADB_API_URL")}
 endpoint:=strings.TrimRight(base,"/")+path
 var body io.Reader
 if data!=nil{b,err:=json.Marshal(data);if err!=nil{return err};body=bytes.NewReader(b)}
 req,err:=http.NewRequestWithContext(ctx,method,endpoint,body);if err!=nil{return err}
 req.Header.Set("Authorization","Bearer "+token)
 if data!=nil{req.Header.Set("Content-Type","application/json")}
 response,err:=(&http.Client{Timeout:25*time.Second}).Do(req)
 if err!=nil{return errors.New("backend request failed")}
 defer response.Body.Close()
 content,err:=io.ReadAll(io.LimitReader(response.Body,4<<20))
 if err!=nil{return errors.New("backend response failed")}
 if response.StatusCode<200||response.StatusCode>=300{return fmt.Errorf("backend returned HTTP %d",response.StatusCode)}
 _,err=out.Write(append(content,'\n'));return err
}
func run(ctx context.Context,args []string,out io.Writer,getenv func(string)string)error{
 if len(args)==0{return errors.New("usage: adbctl connections | schema | query | explain (read-only)")}
 flagset:=flag.NewFlagSet("adbctl",flag.ContinueOnError);flagset.SetOutput(io.Discard)
 id:=flagset.String("id","","database connection ID")
 sql:=flagset.String("sql","","SQL query (SELECT or WITH)")
 raw:=flagset.String("params","[]","bind parameters as JSON array")
 if err:=flagset.Parse(args[1:]);err!=nil{return err}
 base:=getenv("ADB_API_URL");if base==""{base="http://127.0.0.1:8080"}
 token:=getenv("ADB_API_TOKEN")
 action:=args[0]
 switch action{
 case "connections":return call(ctx,"GET","/v1/connections",nil,out,base,token)
 case "schema","query","explain":
  if !connectionID.MatchString(*id){return errors.New("-id must be a 48-character connection ID")}
  path:="/v1/connections/"+*id
  if action=="schema"{return call(ctx,"GET",path+"/schema",nil,out,base,token)}
  if len(*sql)==0||len(*sql)>100000||strings.Contains(*sql,";")||strings.Contains(*sql,"--")||strings.Contains(*sql,"/*")||!regexp.MustCompile(`(?i)^\s*(select|with)\s`).MatchString(*sql){return errors.New("only one SELECT or WITH query is allowed")}
  var params []any
  if err:=json.Unmarshal([]byte(*raw),&params);err!=nil{return errors.New("-params must be a JSON array")}
  if len(params)>100{return errors.New("maximum 100 parameters")}
  for _,p:=range params{switch p.(type){case nil,bool,float64,string:default:return errors.New("only JSON scalar bind parameters are allowed")}}
  return call(ctx,"POST",path+"/"+action,map[string]any{"sql":*sql,"parameters":params,"timeoutSeconds":15},out,base,token)
 default:return errors.New("unknown action: "+action)
 }
}
func main(){if err:=run(context.Background(),os.Args[1:],os.Stdout,os.Getenv);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
