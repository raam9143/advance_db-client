package sshtunnel

import (
 "context"
 "crypto/rand"
 "crypto/rsa"
 "crypto/x509"
 "encoding/pem"
 "fmt"
 "io"
 "net"
 "os"
 "path/filepath"
 "strconv"
 "testing"
 "time"

 "golang.org/x/crypto/ssh"
 "golang.org/x/crypto/ssh/knownhosts"
)
func TestVerifiedSSHLocalForward(t *testing.T){
 echo,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 defer echo.Close()
 go func(){for{c,e:=echo.Accept();if e!=nil{return};go func(){defer c.Close();_,_=io.Copy(c,c)}()}}()
 key,err:=rsa.GenerateKey(rand.Reader,2048);if err!=nil{t.Fatal(err)}
 signer,err:=ssh.NewSignerFromKey(key);if err!=nil{t.Fatal(err)}
 rawKey:=pem.EncodeToMemory(&pem.Block{Type:"RSA PRIVATE KEY",Bytes:x509.MarshalPKCS1PrivateKey(key)})
 bastion,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 defer bastion.Close()
 sshCfg:=&ssh.ServerConfig{PublicKeyCallback:func(_ ssh.ConnMetadata,_ ssh.PublicKey)(*ssh.Permissions,error){return nil,nil}}
 sshCfg.AddHostKey(signer)
 go func(){for{
  conn,e:=bastion.Accept();if e!=nil{return}
  go func(){defer conn.Close()
   _,chans,reqs,e:=ssh.NewServerConn(conn,sshCfg);if e!=nil{return}
   go ssh.DiscardRequests(reqs)
   for ch:=range chans{
    if ch.ChannelType()!="direct-tcpip"{_,_=ch.Reject(ssh.Prohibited,"forwarding refused");continue}
    var req struct{Host string;Port uint32;Origin string;OriginPort uint32}
    if ssh.Unmarshal(ch.ExtraData(),&req)!=nil||req.Host!="127.0.0.1"||req.Port!=uint32(echo.Addr().(*net.TCPAddr).Port){
     _,_=ch.Reject(ssh.Prohibited,"destination refused");continue
    }
    remote,e:=net.DialTimeout("tcp",echo.Addr().String(),3*time.Second);if e!=nil{_,_=ch.Reject(ssh.ConnectionFailed,"destination offline");continue}
    pipe,requests,e:=ch.Accept();if e!=nil{_ =remote.Close();continue}
    go ssh.DiscardRequests(requests)
    go func(){defer remote.Close();defer pipe.Close()
     go func(){_,_=io.Copy(remote,pipe);if c,ok:=remote.(*net.TCPConn);ok{_=c.CloseWrite()}}()
     _,_=io.Copy(pipe,remote)
    }()
   }
  }()
 }}()
 sshAddr:=bastion.Addr().(*net.TCPAddr)
 known:=filepath.Join(t.TempDir(),"known_hosts")
 knownLine:=knownhosts.Line([]string{fmt.Sprintf("[127.0.0.1]:%d",sshAddr.Port)},signer.PublicKey())+"\n"
 if err:=os.WriteFile(known,[]byte(knownLine),0600);err!=nil{t.Fatal(err)}
 targetPort:=echo.Addr().(*net.TCPAddr).Port
 t.Setenv("SSH_TUNNEL_ENABLED","true")
 t.Setenv("SSH_KNOWN_HOSTS_FILE",known)
 t.Setenv("SSH_BASTION_ALLOWLIST",net.JoinHostPort("127.0.0.1",strconv.Itoa(sshAddr.Port)))
 t.Setenv("SSH_DB_TARGET_ALLOWLIST",net.JoinHostPort("127.0.0.1",strconv.Itoa(targetPort)))
 cfg:=Config{Host:"127.0.0.1",Port:sshAddr.Port,User:"tester",PrivateKey:string(rawKey)}
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 tunnel,host,port,err:=Open(ctx,cfg,"127.0.0.1",targetPort)
 if err!=nil{t.Fatal(err)}
 defer tunnel.Close()
 socket,err:=net.DialTimeout("tcp",net.JoinHostPort(host,strconv.Itoa(port)),5*time.Second)
 if err!=nil{t.Fatal(err)}
 defer socket.Close()
 _=socket.SetDeadline(time.Now().Add(5*time.Second))
 sample:=[]byte("hello SSH authenticated TCP forward")
 if _,err=socket.Write(sample);err!=nil{t.Fatal(err)}
 reply:=make([]byte,len(sample))
 if _,err=io.ReadFull(socket,reply);err!=nil{t.Fatal(err)}
 if string(reply)!=string(sample){t.Fatalf("forward corrupted data: %s",reply)}
 // A different pinned host key must fail before a tunnel can be used.
 other,err:=rsa.GenerateKey(rand.Reader,2048);if err!=nil{t.Fatal(err)}
 otherSigner,err:=ssh.NewSignerFromKey(other);if err!=nil{t.Fatal(err)}
 if err=os.WriteFile(known,[]byte(knownhosts.Line([]string{fmt.Sprintf("[127.0.0.1]:%d",sshAddr.Port)},otherSigner.PublicKey())+"\n"),0600);err!=nil{t.Fatal(err)}
 bad,_,_,err:=Open(ctx,cfg,"127.0.0.1",targetPort)
 if err==nil{bad.Close();t.Fatal("SSH host-key mismatch accepted")}
}
