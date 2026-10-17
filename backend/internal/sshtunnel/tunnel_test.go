package sshtunnel
import(
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestRejectUntrustedSSH(t *testing.T){
 t.Setenv("SSH_TUNNEL_ENABLED","false")
 cfg:=Config{Host:"ssh.example.com",Port:22,User:"admin",PrivateKey:"dummy"}
 if Valid(cfg,"127.0.0.1",5432)==nil{t.Fatal("SSH started without opt-in")}
 t.Setenv("SSH_TUNNEL_ENABLED","true")
 t.Setenv("SSH_DB_TARGET_ALLOWLIST","127.0.0.1:5432")
 t.Setenv("SSH_BASTION_ALLOWLIST","ssh.example.com:22")
 file:=filepath.Join(t.TempDir(),"known_hosts")
 if err:=os.WriteFile(file,[]byte("example"),0600);err!=nil{t.Fatal(err)}
 t.Setenv("SSH_KNOWN_HOSTS_FILE",file)
 for _,candidate:=range []struct{host string;port int}{
 {"db.internal",5432},{"127.0.0.1",22},{"127.0.0.1",5433},
 }{if e:=Valid(cfg,candidate.host,candidate.port);e==nil{t.Fatalf("unauthorized target %s:%d",candidate.host,candidate.port)}}
 if err:=Valid(cfg,"127.0.0.1",5432);err!=nil{t.Fatal(err)}
 if err:=os.Chmod(file,0666);err!=nil{t.Fatal(err)}
 if err:=Valid(cfg,"127.0.0.1",5432);err==nil{t.Fatal("writable known_hosts accepted")}
}
func TestRejectInvalidSSHInputs(t *testing.T){
 t.Setenv("SSH_TUNNEL_ENABLED","true")
 t.Setenv("SSH_DB_TARGET_ALLOWLIST","127.0.0.1:5432")
 f:=filepath.Join(t.TempDir(),"known_hosts")
 _=os.WriteFile(f,[]byte("example"),0600);t.Setenv("SSH_KNOWN_HOSTS_FILE",f)
 t.Setenv("SSH_BASTION_ALLOWLIST","localhost:22")
 cfg:=Config{Host:"localhost",Port:22,User:"dbuser",PrivateKey:"key"}
 for _,alter:=range []Config{{Host:"x\nssh",Port:22,User:"u",PrivateKey:"key"},{Host:"localhost",Port:0,User:"u",PrivateKey:"key"},{Host:"localhost",Port:22,User:"",PrivateKey:"key"},{Host:"localhost",Port:22,User:"u",PrivateKey:strings.Repeat("x",65537)}}{
  if err:=Valid(alter,"127.0.0.1",5432);err==nil{t.Fatalf("accepted %+v",alter)}
 }
 if err:=Valid(cfg,"127.0.0.1",5432);err!=nil{t.Fatal(err)}
}
