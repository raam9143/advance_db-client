// Package vault provides operator-managed, authenticated encryption for named DB profiles.
// Private credentials are never returned by the web API or stored in browser state.
package vault

import (
 "crypto/aes"
 "crypto/cipher"
 "crypto/rand"
 "crypto/sha256"
 "encoding/base64"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "regexp"
 "strings"
)
const maxSize=1<<20
var validName=regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
type Credentials struct {
 Engine string `json:"engine"`
 Host string `json:"host"`
 Port int `json:"port"`
 Database string `json:"database"`
 User string `json:"user"`
 Password string `json:"password"`
 SSLMode string `json:"sslMode"`
 ServiceMode string `json:"serviceMode,omitempty"`
}
type sealed struct {
 KeyID string `json:"keyId"`
 Nonce string `json:"nonce"`
 Data string `json:"data"`
}
type state struct{
 Version int `json:"version"`
 Records map[string]sealed `json:"records"`
}
type Store struct{
 path string
 active []byte
 keys map[string][]byte
}
func decodeKey(value string)([]byte,error){
 key,err:=base64.StdEncoding.Strict().DecodeString(value)
 if err!=nil||len(key)!=32{return nil,errors.New("vault key must be exactly 32 random bytes in standard Base64")}
 return key,nil
}
func keyID(key []byte)string{
 h:=sha256.Sum256(append([]byte("adb-vault-kid:"),key...))
 return hex.EncodeToString(h[:12])
}
func New(path,activeB64 string,previousB64 []string)(*Store,error){
 if !filepath.IsAbs(path){return nil,errors.New("vault path must be absolute")}
 directory:=filepath.Dir(path)
 stat,err:=os.Lstat(directory)
 if err!=nil||!stat.IsDir()||stat.Mode()&os.ModeSymlink!=0||stat.Mode().Perm()&0077!=0{return nil,errors.New("vault directory must exist with 0700 permissions")}
 active,err:=decodeKey(activeB64);if err!=nil{return nil,err}
 keys:=map[string][]byte{keyID(active):active}
 for _,b64:=range previousB64{
  if strings.TrimSpace(b64)==""{continue}
  key,err:=decodeKey(b64);if err!=nil{return nil,err}
  keys[keyID(key)]=key
 }
 if len(keys)>5{return nil,errors.New("vault accepts at most four previous keys")}
 return &Store{path:path,active:active,keys:keys},nil
}
func FromEnv()(*Store,error){
 path:=os.Getenv("VAULT_FILE")
 if path==""{return nil,errors.New("VAULT_FILE is not configured")}
 previous:=strings.Split(os.Getenv("VAULT_PREVIOUS_KEYS_B64"),",")
 return New(path,os.Getenv("VAULT_ACTIVE_KEY_B64"),previous)
}
func(s *Store)read()(state,error){
 a:=state{Version:1,Records:map[string]sealed{}}
 stat,err:=os.Lstat(s.path)
 if os.IsNotExist(err){return a,nil}
 if err!=nil{return a,err}
 if !stat.Mode().IsRegular()||stat.Mode().Perm()&0077!=0||stat.Size()>maxSize{return a,errors.New("vault file is unsafe")}
 f,err:=os.Open(s.path);if err!=nil{return a,err};defer f.Close()
 data,err:=io.ReadAll(io.LimitReader(f,maxSize+1));if err!=nil{return a,err}
 if len(data)>maxSize{return a,errors.New("vault too large")}
 if err=json.Unmarshal(data,&a);err!=nil||a.Version!=1||a.Records==nil||len(a.Records)>100{return a,errors.New("vault file format invalid")}
 return a,nil
}
func(s *Store)seal(name string,c Credentials)(sealed,error){
 var result sealed
 plain,err:=json.Marshal(c);if err!=nil{return result,err}
 if len(plain)>8192{return result,errors.New("credentials too large")}
 b,err:=aes.NewCipher(s.active);if err!=nil{return result,err}
 aead,err:=cipher.NewGCM(b);if err!=nil{return result,err}
 nonce:=make([]byte,aead.NonceSize())
 if _,err=rand.Read(nonce);err!=nil{return result,err}
 kid:=keyID(s.active)
 protected:=aead.Seal(nil,nonce,plain,[]byte("adb-vault:v1:"+name+":"+kid))
 return sealed{KeyID:kid,Nonce:base64.StdEncoding.EncodeToString(nonce),Data:base64.StdEncoding.EncodeToString(protected)},nil
}
func(s *Store)open(name string,record sealed)(Credentials,error){
 var cred Credentials
 key,found:=s.keys[record.KeyID]
 if !found{return cred,errors.New("vault record requires unavailable rotation key")}
 nonce,err:=base64.StdEncoding.Strict().DecodeString(record.Nonce)
 if err!=nil{return cred,errors.New("vault nonce invalid")}
 ciphertext,err:=base64.StdEncoding.Strict().DecodeString(record.Data)
 if err!=nil||len(ciphertext)>8192{return cred,errors.New("vault ciphertext invalid")}
 block,err:=aes.NewCipher(key);if err!=nil{return cred,err}
 aead,err:=cipher.NewGCM(block);if err!=nil{return cred,err}
 if len(nonce)!=aead.NonceSize(){return cred,errors.New("vault nonce length invalid")}
 plain,err:=aead.Open(nil,nonce,ciphertext,[]byte("adb-vault:v1:"+name+":"+record.KeyID))
 if err!=nil{return cred,errors.New("vault integrity check failed")}
 if err=json.Unmarshal(plain,&cred);err!=nil{return cred,errors.New("vault credential JSON invalid")}
 if cred.Engine==""||cred.Database==""||len(cred.Password)>2048{return cred,errors.New("vault credential invalid")}
 return cred,nil
}
func(s *Store)Get(name string)(Credentials,error){
 var zero Credentials
 if !validName.MatchString(name){return zero,errors.New("invalid vault credential name")}
 a,err:=s.read();if err!=nil{return zero,err}
 record,ok:=a.Records[name]
 if !ok{return zero,errors.New("credential not found")}
 return s.open(name,record)
}
func(s *Store)List()([]string,error){
 a,err:=s.read();if err!=nil{return nil,err}
 names:=make([]string,0,len(a.Records))
 for name:=range a.Records{names=append(names,name)}
 return names,nil
}
func(s *Store)write(a state)error{
 raw,err:=json.MarshalIndent(a,"","  ");if err!=nil{return err}
 if len(raw)>maxSize{return errors.New("vault exceeds maximum size")}
 dir:=filepath.Dir(s.path)
 f,err:=os.CreateTemp(dir,".adb-vault-*")
 if err!=nil{return err}
 tmp:=f.Name();defer os.Remove(tmp)
 if err=f.Chmod(0600);err!=nil{_ =f.Close();return err}
 if _,err=f.Write(raw);err!=nil{_ =f.Close();return err}
 if err=f.Sync();err!=nil{_ =f.Close();return err}
 if err=f.Close();err!=nil{return err}
 if err=os.Rename(tmp,s.path);err!=nil{return err}
 folder,err:=os.Open(dir);if err==nil{_=folder.Sync();_=folder.Close()}
 return nil
}
// Put and Rotate are offline operator operations; do not run two writers concurrently.
func(s *Store)Put(name string,c Credentials)error{
 if !validName.MatchString(name){return errors.New("invalid credential name")}
 if c.Engine==""||c.Database==""||len(c.Password)>2048{return errors.New("invalid credential fields")}
 a,err:=s.read();if err!=nil{return err}
 if len(a.Records)>=100{if _,ok:=a.Records[name];!ok{return errors.New("vault record limit reached")}}
 entry,err:=s.seal(name,c);if err!=nil{return err}
 a.Records[name]=entry
 return s.write(a)
}
func(s *Store)Rotate()(int,error){
 a,err:=s.read();if err!=nil{return 0,err}
 changed:=make(map[string]sealed,len(a.Records))
 for name,item:=range a.Records{
  c,err:=s.open(name,item);if err!=nil{return 0,fmt.Errorf("cannot rotate %q: %w",name,err)}
  x,err:=s.seal(name,c);if err!=nil{return 0,err}
  changed[name]=x
 }
 a.Records=changed
 if err=s.write(a);err!=nil{return 0,err}
 return len(changed),nil
}
