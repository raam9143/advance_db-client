// Package sshtunnel starts a bounded, host-key-verified TCP forward for an SSH-local database.
package sshtunnel

import (
 "context"
 "errors"
 "io"
 "net"
 "os"
 "path/filepath"
 "strconv"
 "strings"
 "sync"
 "time"

 "golang.org/x/crypto/ssh"
 "golang.org/x/crypto/ssh/knownhosts"
)
type Config struct{
 Host string `json:"host"`
 Port int `json:"port"`
 User string `json:"user"`
 PrivateKey string `json:"privateKey"`
}
type Tunnel struct{
 client *ssh.Client
 listener net.Listener
 done chan struct{}
 once sync.Once
 limit chan struct{}
}
func Valid(c Config,databaseHost string,databasePort int)error{
 if os.Getenv("SSH_TUNNEL_ENABLED")!="true"{return errors.New("SSH tunneling disabled")}
 if c.Host==""||len(c.Host)>253||strings.ContainsAny(c.Host,"/\\\x00\r\n")||c.Port<1||c.Port>65535||c.User==""||len(c.User)>128{return errors.New("invalid SSH server")}
 if !strings.EqualFold(databaseHost,"localhost")&&databaseHost!="127.0.0.1"&&databaseHost!="::1"{return errors.New("SSH tunnel target must be loopback on SSH server")}
 if databasePort<1||databasePort>65535{return errors.New("invalid SSH database port")}
 if len(c.PrivateKey)==0||len(c.PrivateKey)>65536{return errors.New("SSH private key required (max 64 KiB)")}
 target:=net.JoinHostPort(databaseHost,strconv.Itoa(databasePort))
 allowed:=false
 for _,item:=range strings.Split(os.Getenv("SSH_DB_TARGET_ALLOWLIST"),","){if strings.TrimSpace(item)==target{allowed=true;break}}
 if !allowed{return errors.New("SSH database port not allowlisted")}
 bastion:=net.JoinHostPort(c.Host,strconv.Itoa(c.Port))
 bastionAllowed:=false
 for _,item:=range strings.Split(os.Getenv("SSH_BASTION_ALLOWLIST"),","){if strings.TrimSpace(item)==bastion{bastionAllowed=true;break}}
 if !bastionAllowed{return errors.New("SSH bastion not allowlisted")}
 known:=os.Getenv("SSH_KNOWN_HOSTS_FILE")
 if !filepath.IsAbs(known){return errors.New("SSH_KNOWN_HOSTS_FILE must be an absolute path")}
 st,err:=os.Lstat(known);if err!=nil||!st.Mode().IsRegular(){return errors.New("SSH known hosts file missing or invalid")}
 if st.Mode().Perm()&0022!=0{return errors.New("SSH known hosts must not be group/world writable")}
 return nil
}
func Open(ctx context.Context,c Config,databaseHost string,databasePort int)(*Tunnel,string,int,error){
 if err:=Valid(c,databaseHost,databasePort);err!=nil{return nil,"",0,err}
 key,err:=ssh.ParsePrivateKey([]byte(c.PrivateKey));if err!=nil{return nil,"",0,errors.New("SSH private key invalid")}
 check,err:=knownhosts.New(os.Getenv("SSH_KNOWN_HOSTS_FILE"));if err!=nil{return nil,"",0,errors.New("SSH known hosts invalid")}
 dest:=net.JoinHostPort(c.Host,strconv.Itoa(c.Port))
 socket,err:=(&net.Dialer{Timeout:8*time.Second}).DialContext(ctx,"tcp",dest)
 if err!=nil{return nil,"",0,errors.New("SSH connection failed")}
 conf:=&ssh.ClientConfig{User:c.User,Auth:[]ssh.AuthMethod{ssh.PublicKeys(key)},HostKeyCallback:check,Timeout:8*time.Second,Config:ssh.Config{Ciphers:[]string{"aes256-gcm@openssh.com","aes128-gcm@openssh.com","chacha20-poly1305@openssh.com"}}}
 hand,channels,requests,err:=ssh.NewClientConn(socket,dest,conf)
 if err!=nil{_ =socket.Close();return nil,"",0,errors.New("SSH authentication or host-key verification failed")}
 client:=ssh.NewClient(hand,channels,requests)
 listener,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil{_ =client.Close();return nil,"",0,errors.New("SSH local listener failed")}
 t:=&Tunnel{client:client,listener:listener,done:make(chan struct{}),limit:make(chan struct{},8)}
 target:=net.JoinHostPort(databaseHost,strconv.Itoa(databasePort))
 go t.accept(target)
 addr:=listener.Addr().(*net.TCPAddr)
 return t,"127.0.0.1",addr.Port,nil
}
func(t *Tunnel)Close(){
 t.once.Do(func(){close(t.done);_ =t.listener.Close();_ =t.client.Close()})
}
func(t *Tunnel)accept(target string){
 for {
  source,err:=t.listener.Accept()
  if err!=nil{return}
  select{
  case <-t.done:_=source.Close();return
  case t.limit<-struct{}{}:
    go func(){defer func(){<-t.limit}();t.forward(source,target)}()
  default:_=source.Close()
  }
 }
}
func(t *Tunnel)forward(source net.Conn,target string){
 defer source.Close()
 destination,err:=t.client.Dial("tcp",target)
 if err!=nil{return}
 defer destination.Close()
 var group sync.WaitGroup
 group.Add(2)
 go func(){defer group.Done();_,_=io.Copy(destination,source);if x,ok:=destination.(interface{CloseWrite()error});ok{_=x.CloseWrite()}}()
 go func(){defer group.Done();_,_=io.Copy(source,destination);if x,ok:=source.(interface{CloseWrite()error});ok{_=x.CloseWrite()}}()
 group.Wait()
}
