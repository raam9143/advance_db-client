// Package redisclient implements a bounded, allowlisted, read-only RESP2 inspector.
package redisclient

import (
 "bufio"
 "context"
 "crypto/tls"
 "errors"
 "fmt"
 "io"
 "net"
 "regexp"
 "strconv"
 "strings"
 "time"
)

const (maxValue=1<<20;maxResponse=4<<20;maxArray=256)
type Config struct {
 Host string
 Port int
 Username string
 Password string
 Database int
 // PlaintextLoopback must only be enabled explicitly for local development.
 PlaintextLoopback bool
}
type Client struct{config Config}
type Request struct{
 Command string `json:"command"`
 Key string `json:"key,omitempty"`
 Cursor string `json:"cursor,omitempty"`
 Pattern string `json:"pattern,omitempty"`
 Start int `json:"start,omitempty"`
 Stop int `json:"stop,omitempty"`
}
type Result struct{
 Command string `json:"command"`
 Value any `json:"value"`
}
var digits=regexp.MustCompile(`^[0-9]{1,20}$`)
func New(c Config)(*Client,error){
 if c.Host==""||len(c.Host)>253||strings.ContainsAny(c.Host,"/\\\x00\n\r")||c.Port<=0||c.Port>65535 {return nil,errors.New("invalid Redis endpoint")}
 if c.Database<0||c.Database>15{return nil,errors.New("Redis database must be 0-15")}
 if len(c.Username)>128||len(c.Password)>512{return nil,errors.New("Redis authentication field too long")}
 if c.PlaintextLoopback&&!loopback(c.Host){return nil,errors.New("plaintext Redis requires loopback address")}
 if c.Password==""&&c.Username!=""{return nil,errors.New("Redis ACL username requires password")}
 return &Client{config:c},nil
}
func loopback(host string)bool{if strings.EqualFold(host,"localhost"){return true};ip:=net.ParseIP(host);return ip!=nil&&ip.IsLoopback()}
func(c *Client)dial(ctx context.Context)(net.Conn,error){
 dest:=net.JoinHostPort(c.config.Host,strconv.Itoa(c.config.Port))
 d:=net.Dialer{Timeout:5*time.Second}
 if c.config.PlaintextLoopback{return d.DialContext(ctx,"tcp",dest)}
 return (&tls.Dialer{NetDialer:&d,Config:&tls.Config{MinVersion:tls.VersionTLS12}}).DialContext(ctx,"tcp",dest)
}
func(c *Client)withConn(ctx context.Context,fn func(*wire)(any,error))(any,error){
 conn,err:=c.dial(ctx);if err!=nil{return nil,errors.New("Redis connection failed")}
 defer conn.Close()
 _=conn.SetDeadline(time.Now().Add(10*time.Second))
 stop:=context.AfterFunc(ctx,func(){_=conn.SetDeadline(time.Now())});defer stop()
 w:=&wire{conn:conn,rd:bufio.NewReaderSize(conn,16*1024),budget:maxResponse}
 if c.config.Password!="" {
  auth:=[]string{"AUTH",c.config.Password};if c.config.Username!=""{auth=[]string{"AUTH",c.config.Username,c.config.Password}}
  if _,err=w.call(auth);err!=nil{return nil,errors.New("Redis authentication failed")}
 }
 if c.config.Database!=0{if _,err=w.call([]string{"SELECT",strconv.Itoa(c.config.Database)});err!=nil{return nil,errors.New("Redis database selection failed")}}
 return fn(w)
}
func(c *Client)Ping(ctx context.Context)error{
 v,e:=c.withConn(ctx,func(w *wire)(any,error){return w.call([]string{"PING"})})
 if e!=nil{return e};if v!="PONG"{return errors.New("invalid Redis PING response")};return nil
}
func(c *Client)Inspect(ctx context.Context,r Request)(Result,error){
 cmd:=strings.ToUpper(strings.TrimSpace(r.Command));result:=Result{Command:cmd}
 var argv []string
 if len(r.Key)>1024||len(r.Pattern)>256{return result,errors.New("Redis key or pattern too long")}
 switch cmd {
 case "GET","TYPE","TTL","HGETALL":
  if r.Key==""{return result,errors.New("Redis key is required")}
  argv=[]string{cmd,r.Key}
 case "LRANGE":
  if r.Key==""||r.Start<0||r.Start>1000000||r.Stop<r.Start||r.Stop-r.Start>99{return result,errors.New("LRANGE must specify at most 100 items")}
  argv=[]string{cmd,r.Key,strconv.Itoa(r.Start),strconv.Itoa(r.Stop)}
 case "SCAN":
  cursor:=r.Cursor;if cursor==""{cursor="0"}
  if !digits.MatchString(cursor){return result,errors.New("invalid Redis cursor")}
  argv=[]string{"SCAN",cursor,"MATCH","*","COUNT","50"}
  if r.Pattern!=""{argv[3]=r.Pattern}
 default:return result,errors.New("unsupported Redis read command")
 }
 v,e:=c.withConn(ctx,func(w *wire)(any,error){return w.call(argv)})
 if e!=nil{return result,e}
 result.Value=v;return result,nil
}
type wire struct{conn net.Conn;rd *bufio.Reader;budget int}
func(w *wire)call(args []string)(any,error){
 var b strings.Builder
 fmt.Fprintf(&b,"*%d\r\n",len(args))
 for _,a:=range args{fmt.Fprintf(&b,"$%d\r\n%s\r\n",len(a),a)}
 if _,err:=io.WriteString(w.conn,b.String());err!=nil{return nil,errors.New("Redis write failed")}
 return w.reply(0)
}
func(w *wire)line()(string,error){
 line,err:=w.rd.ReadSlice('\n')
 if err!=nil{return "",errors.New("Redis protocol line too long")}
 if len(line)<2||line[len(line)-2]!='\r'||len(line)>16*1024{return "",errors.New("invalid Redis protocol line")}
 w.budget-=len(line);if w.budget<0{return "",errors.New("Redis response too large")}
 return string(line[:len(line)-2]),nil
}
func(w *wire)reply(depth int)(any,error){
 if depth>3{return nil,errors.New("Redis nested response too deep")}
 line,err:=w.line();if err!=nil{return nil,err};if len(line)==0{return nil,errors.New("empty Redis reply")}
 switch line[0] {
 case '+':return line[1:],nil
 case '-':return nil,errors.New("Redis command rejected")
 case ':':n,e:=strconv.ParseInt(line[1:],10,64);if e!=nil{return nil,errors.New("invalid Redis integer")};return n,nil
 case '$':
  n,e:=strconv.Atoi(line[1:]);if e!=nil||n < -1||n>maxValue{return nil,errors.New("invalid Redis bulk length")}
  if n==-1{return nil,nil};if n+2>w.budget{return nil,errors.New("Redis response limit exceeded")}
  b:=make([]byte,n+2);if _,e=io.ReadFull(w.rd,b);e!=nil{return nil,errors.New("incomplete Redis response")}
  if b[n]!='\r'||b[n+1]!='\n'{return nil,errors.New("invalid Redis bulk terminator")}
  w.budget-=n+2;return string(b[:n]),nil
 case '*':
  n,e:=strconv.Atoi(line[1:]);if e!=nil||n < -1||n>maxArray{return nil,errors.New("Redis array too large")}
  if n==-1{return nil,nil}
  items:=make([]any,n);for i:=range items{items[i],e=w.reply(depth+1);if e!=nil{return nil,e}}
  return items,nil
 default:return nil,errors.New("unsupported Redis response type")
 }
}
