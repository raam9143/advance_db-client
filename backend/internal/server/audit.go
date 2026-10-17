package server

import(
 "bufio"
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "time"
)
type auditEntry struct{
 Time string `json:"timestamp"`
 Method string `json:"method"`
 PathHash string `json:"pathHash"`
 OwnerHash string `json:"ownerHash"`
 Event string `json:"event"`
 Status int `json:"status"`
 Previous string `json:"previous"`
 MAC string `json:"mac"`
}
type auditLog struct{
 file *os.File
 key []byte
 previous string
 mu sync.Mutex
}
type auditResponseWriter struct{http.ResponseWriter;status int}
func(w *auditResponseWriter)WriteHeader(code int){w.status=code;w.ResponseWriter.WriteHeader(code)}
func(w *auditResponseWriter)Write(body []byte)(int,error){return w.ResponseWriter.Write(body)}
func digest(key []byte,data string)string{
 mac:=hmac.New(sha256.New,key);mac.Write([]byte(data))
 return hex.EncodeToString(mac.Sum(nil))
}
func signAudit(key []byte,e auditEntry)string{
 e.MAC=""
 raw,_:=json.Marshal(e)
 return digest(key,string(raw))
}
func openAuditLog()(*auditLog,error){
 path:=os.Getenv("AUDIT_LOG_PATH")
 if path==""{return nil,nil}
 key:=os.Getenv("AUDIT_HMAC_KEY")
 if len(key)<32{return nil,errors.New("AUDIT_HMAC_KEY must have at least 32 bytes")}
 if !filepath.IsAbs(path){return nil,errors.New("AUDIT_LOG_PATH must be absolute")}
 parent:=filepath.Dir(path)
 st,err:=os.Lstat(parent)
 if err!=nil||!st.IsDir()||st.Mode()&os.ModeSymlink!=0||st.Mode().Perm()&0077!=0{return nil,errors.New("audit directory must exist with 0700 permissions")}
 st,err=os.Lstat(path)
 if err==nil{
  if !st.Mode().IsRegular()||st.Mode().Perm()&0077!=0{return nil,errors.New("audit file must be regular and owner-private")}
 }else if !os.IsNotExist(err){return nil,err}
 file,err:=os.OpenFile(path,os.O_RDWR|os.O_APPEND|os.O_CREATE,0600)
 if err!=nil{return nil,err}
 a:=&auditLog{file:file,key:[]byte(key)}
 if err=a.checkExisting();err!=nil{_ =file.Close();return nil,err}
 return a,nil
}
func(a *auditLog)checkExisting()error{
 stat,err:=a.file.Stat()
 if err!=nil{return err}
 if stat.Size()>100<<20{return errors.New("audit log exceeds maximum size; rotate offline")}
 if _,err=a.file.Seek(0,io.SeekStart);err!=nil{return err}
 scanner:=bufio.NewScanner(a.file)
 scanner.Buffer(make([]byte,16*1024),1<<20)
 previous:=""
 for scanner.Scan(){
  var e auditEntry
  if err=json.Unmarshal(scanner.Bytes(),&e);err!=nil{return errors.New("audit log contains invalid JSON")}
  if e.Previous!=previous||e.MAC!=signAudit(a.key,e)||len(e.MAC)!=64{return errors.New("audit log integrity verification failed")}
  previous=e.MAC
 }
 if err=scanner.Err();err!=nil{return err}
 a.previous=previous
 _,err=a.file.Seek(0,io.SeekEnd)
 return err
}
func(a *auditLog)Append(method,path,owner,event string,status int)error{
 a.mu.Lock();defer a.mu.Unlock()
 if !strings.Contains(" GET POST PUT PATCH DELETE HEAD OPTIONS "," "+method+" "){method="UNKNOWN"}
 if len(path)>1024{path=path[:1024]}
 e:=auditEntry{
  Time:time.Now().UTC().Format(time.RFC3339Nano),
  Method:method,PathHash:digest(a.key,"path:"+path),OwnerHash:digest(a.key,"owner:"+owner),
  Event:event,Status:status,Previous:a.previous,
 }
 e.MAC=signAudit(a.key,e)
 b,err:=json.Marshal(e);if err!=nil{return err}
 b=append(b,'\n')
 if _,err=a.file.Write(b);err!=nil{return err}
 if err=a.file.Sync();err!=nil{return err}
 a.previous=e.MAC
 return nil
}
