// Package prodcheck enforces a minimum runtime security profile before listening.
package prodcheck

import(
 "encoding/base64"
 "github.com/raam9143/advance_db-client/backend/internal/egress"
 "errors"
 "fmt"
 "net/url"
 "os"
 "strings"
)
func Validate(get func(string)string)error{
 if get("ADB_PRODUCTION")!="true"{return nil}
 errorsList:=[]string{}
 require:=func(k string){if get(k)==""{errorsList=append(errorsList,k+" required")}}
 require("API_TOKEN");require("AUDIT_LOG_PATH");require("AUDIT_HMAC_KEY");require("AUDIT_SINK_URL");require("AUDIT_SINK_TOKEN")
 require("VAULT_FILE");require("VAULT_ACTIVE_KEY_B64")
 if err:=egress.ValidateList(get("DB_ENDPOINT_ALLOWLIST"));err!=nil{errorsList=append(errorsList,err.Error())}
 token:=get("API_TOKEN")
 if len(token)<32{errorsList=append(errorsList,"API_TOKEN must be at least 32 characters")}
 if token!=""&&strings.Trim(token,string(token[0]))==""{errorsList=append(errorsList,"API_TOKEN cannot be a repeated character")}
 if len(get("AUDIT_HMAC_KEY"))<32{errorsList=append(errorsList,"AUDIT_HMAC_KEY must be at least 32 characters")}
 if len(get("AUDIT_SINK_TOKEN"))<32{errorsList=append(errorsList,"AUDIT_SINK_TOKEN must be at least 32 characters")}
 sink,err:=url.Parse(get("AUDIT_SINK_URL"))
 if err!=nil||sink.Scheme!="https"||sink.Hostname()==""||sink.User!=nil||sink.RawQuery!=""||sink.Fragment!=""{errorsList=append(errorsList,"AUDIT_SINK_URL must be HTTPS without inline credentials")}
 key,err:=base64.StdEncoding.Strict().DecodeString(get("VAULT_ACTIVE_KEY_B64"))
 if err!=nil||len(key)!=32{errorsList=append(errorsList,"VAULT_ACTIVE_KEY_B64 must encode exactly 32 random bytes")}
 for _,name:=range []string{"MONGO_ALLOW_PLAINTEXT_LOOPBACK","REDIS_ALLOW_PLAINTEXT_LOOPBACK","SQLSERVER_ALLOW_UNVERIFIED_LOOPBACK","ALLOW_WRITE_QUERIES"}{
  if get(name)=="true"{errorsList=append(errorsList,name+" prohibited in certified read-only profile")}
 }
 if get("SSH_TUNNEL_ENABLED")=="true"&&get("SSH_KNOWN_HOSTS_FILE")==""{errorsList=append(errorsList,"SSH_KNOWN_HOSTS_FILE required")}
 if strings.HasPrefix(get("VAULT_FILE"),"http")||!strings.HasPrefix(get("VAULT_FILE"),"/"){errorsList=append(errorsList,"VAULT_FILE must be absolute local path")}
 if get("AI_ASSIST_ENABLED")=="true"&&get("AI_DATA_SHARING_APPROVED")!="true"{errorsList=append(errorsList,"AI requires prior data-sharing approval")}
 if listen:=get("API_LISTEN");listen!=""{
  if strings.Contains(listen,"http"){errorsList=append(errorsList,"API_LISTEN must be a host:port")}
  _,err=url.Parse("http://"+listen);if err!=nil{errorsList=append(errorsList,"invalid API listen address")}
 }
 if len(errorsList)>0{return fmt.Errorf("production profile refused: %s",strings.Join(errorsList,"; "))}
 return nil
}
func ValidateEnv()error{return Validate(os.Getenv)}
var ErrNotProduction=errors.New("not running in production profile")
