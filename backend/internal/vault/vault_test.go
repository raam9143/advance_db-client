package vault
import(
 "crypto/rand"
 "encoding/base64"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func randomKey(t *testing.T)string{
 t.Helper();key:=make([]byte,32);if _,e:=rand.Read(key);e!=nil{t.Fatal(e)}
 return base64.StdEncoding.EncodeToString(key)
}
func TestVaultSecrecyAndRotation(t *testing.T){
 dir:=t.TempDir();if err:=os.Chmod(dir,0700);err!=nil{t.Fatal(err)}
 file:=filepath.Join(dir,"db.enc")
 oldKey,newKey:=randomKey(t),randomKey(t)
 old,e:=New(file,oldKey,nil);if e!=nil{t.Fatal(e)}
 initial:=Credentials{Engine:"postgres",Host:"db.internal",Port:5432,Database:"prod",User:"readonly",Password:"VERY_SENSITIVE_PASSWORD"}
 if e=old.Put("my_db",initial);e!=nil{t.Fatal(e)}
 raw,e:=os.ReadFile(file);if e!=nil{t.Fatal(e)}
 if strings.Contains(string(raw),initial.Password)||strings.Contains(string(raw),initial.Host){t.Fatal("secret persisted in cleartext")}
 if _,e=old.Get("my_db");e!=nil{t.Fatal(e)}
 if _,e=old.Get("other");e==nil{t.Fatal("arbitrary credential lookup")}
 wrong,e:=New(file,newKey,nil);if e!=nil{t.Fatal(e)}
 if _,e=wrong.Get("my_db");e==nil{t.Fatal("vault decrypted with unrelated key")}
 rotated,e:=New(file,newKey,[]string{oldKey});if e!=nil{t.Fatal(e)}
 n,e:=rotated.Rotate();if e!=nil||n!=1{t.Fatalf("rotate: %d %v",n,e)}
 current,e:=New(file,newKey,nil);if e!=nil{t.Fatal(e)}
 after,e:=current.Get("my_db")
 if e!=nil||after.Password!=initial.Password{t.Fatalf("new key failed: %v",e)}
 if _,e=old.Get("my_db");e==nil{t.Fatal("old key still decrypts rotated vault")}
}
func TestVaultFileAndRecordTamper(t *testing.T){
 dir:=t.TempDir();_ =os.Chmod(dir,0700)
 file:=filepath.Join(dir,"records")
 v,e:=New(file,randomKey(t),nil);if e!=nil{t.Fatal(e)}
 _=v.Put("connection",Credentials{Engine:"sqlite",Database:"prod",Password:"abc"})
 raw,e:=os.ReadFile(file);if e!=nil{t.Fatal(e)}
 raw[len(raw)/2]^=1
 if e=os.WriteFile(file,raw,0600);e!=nil{t.Fatal(e)}
 if _,e=v.Get("connection");e==nil{t.Fatal("ciphertext tampering accepted")}
 if e=os.Chmod(file,0644);e!=nil{t.Fatal(e)}
 if _,e=v.List();e==nil{t.Fatal("unsafe permissions accepted")}
}
func TestVaultRejectInvalidKeyAndDirectory(t *testing.T){
 dir:=t.TempDir()
 _=os.Chmod(dir,0755)
 if _,e:=New(filepath.Join(dir,"vault"),randomKey(t),nil);e==nil{t.Fatal("unsafe directory accepted")}
 _=os.Chmod(dir,0700)
 if _,e:=New(filepath.Join(dir,"vault"),"bad",nil);e==nil{t.Fatal("invalid key accepted")}
}
