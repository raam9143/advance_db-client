// Package egress implements an explicit per-endpoint network policy in production.
// Enforce matching firewall rules too: DNS can change after admission.
package egress

import(
 "errors"
 "net"
 "os"
 "strconv"
 "strings"
)
func approvedEntry(value string)bool{
 if strings.TrimSpace(value)!=value||value==""||strings.ContainsAny(value,"/*?@#\\\r\n\t"){return false}
 host,portText,err:=net.SplitHostPort(value)
 if err!=nil||host==""||strings.Contains(host,"%"){return false}
 port,err:=strconv.Atoi(portText)
 if err!=nil||port<1||port>65535{return false}
 if strings.ContainsAny(host," []"){return false}
 ip:=net.ParseIP(host)
 if ip!=nil{
  if ip.IsLinkLocalUnicast()||ip.IsLinkLocalMulticast()||ip.IsUnspecified()||ip.IsMulticast(){return false}
 }else{
  if strings.HasPrefix(host,"-")||strings.HasSuffix(host,"-")||strings.HasSuffix(host,"."){return false}
  for _,c:=range host{if !((c>='a'&&c<='z')||(c>='A'&&c<='Z')||(c>='0'&&c<='9')||c=='-'||c=='.'){return false}}
 }
 return true
}
func ValidateList(list string)error{
 if list==""{return errors.New("DB_ENDPOINT_ALLOWLIST must be configured")}
 entries:=strings.Split(list,",")
 if len(entries)>100{return errors.New("too many approved database endpoints")}
 for _,item:=range entries{if !approvedEntry(item){return errors.New("invalid database endpoint in allowlist")}}
 return nil
}
func Allowed(host string,port int)bool{
 if os.Getenv("ADB_PRODUCTION")!="true"{return true}
 list:=os.Getenv("DB_ENDPOINT_ALLOWLIST")
 if ValidateList(list)!=nil||port<1||port>65535{return false}
 sought:=net.JoinHostPort(strings.ToLower(host),strconv.Itoa(port))
 for _,entry:=range strings.Split(list,","){
  eh,ep,err:=net.SplitHostPort(entry)
  if err==nil&&sought==net.JoinHostPort(strings.ToLower(eh),ep){return true}
 }
 return false
}
