package server
import(
 "context"
 "crypto/rand"
 "encoding/hex"
 "errors"
 "fmt"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "time"
)
const(maxBackupBytes=128<<20;backupTTL=30*time.Minute)
type backupSnapshot struct{owner string;path string;size int64;expires time.Time}
func privateBackupRoot()(string,error){
 root:=os.Getenv("SQLITE_BACKUP_DIR")
 if root==""||!filepath.IsAbs(root){return "",errors.New("SQLite snapshots are disabled (set SQLITE_BACKUP_DIR)")}
 info,err:=os.Lstat(root)
 if err!=nil||!info.IsDir()||info.Mode()&os.ModeSymlink!=0{return "",errors.New("invalid snapshot root")}
 if info.Mode().Perm()&0077!=0{return "",errors.New("snapshot root must have permissions 0700")}
 return root,nil
}
func(s *Server)createBackup(w http.ResponseWriter,r *http.Request,id string){
 c,ok:=s.lookup(id,requestOwner(r))
 if !ok{fail(w,404,"connection not found");return}
 if c.info.Engine!="sqlite"||c.db==nil{fail(w,400,"only SQLite connections support snapshots");return}
 var request struct{ConfirmBackup bool `json:"confirmBackup"`}
 if !decode(w,r,&request){return}
 if !request.ConfirmBackup{fail(w,400,"explicit backup confirmation required");return}
 root,err:=privateBackupRoot()
 if err!=nil{fail(w,503,err.Error());return}
 ctx,cancel:=context.WithTimeout(r.Context(),25*time.Second);defer cancel()
 var pages,pageSize int64
 if e:=c.db.QueryRowContext(ctx,"PRAGMA page_count").Scan(&pages);e!=nil{fail(w,400,"could not read SQLite page count");return}
 if e:=c.db.QueryRowContext(ctx,"PRAGMA page_size").Scan(&pageSize);e!=nil{fail(w,400,"could not read SQLite page size");return}
 if pages<0||pageSize<=0||pages>maxBackupBytes/pageSize{fail(w,413,"SQLite database exceeds snapshot limit");return}
 var random [24]byte
 if _,err=rand.Read(random[:]);err!=nil{fail(w,500,"backup ID generation failed");return}
 backupID:=hex.EncodeToString(random[:])
 dir,err:=os.MkdirTemp(root,"adb-snapshot-")
 if err!=nil{fail(w,500,"snapshot directory creation failed");return}
 path:=filepath.Join(dir,"snapshot.sqlite")
 if _,err=c.db.ExecContext(ctx,"VACUUM main INTO ?",path);err!=nil{
  _=os.RemoveAll(dir);fail(w,400,"SQLite snapshot failed");return
 }
 stat,err:=os.Lstat(path)
 if err!=nil||!stat.Mode().IsRegular()||stat.Size()>maxBackupBytes{
  _=os.RemoveAll(dir);fail(w,400,"snapshot exceeds maximum size");return
 }
 if err=os.Chmod(path,0600);err!=nil{_=os.RemoveAll(dir);fail(w,500,"cannot secure snapshot");return}
 s.mu.Lock()
 if len(s.backups)>=10{s.mu.Unlock();_=os.RemoveAll(dir);fail(w,429,"snapshot limit reached");return}
 expires:=s.now().Add(backupTTL)
 s.backups[backupID]=&backupSnapshot{owner:requestOwner(r),path:path,size:stat.Size(),expires:expires}
 s.mu.Unlock()
 respond(w,201,map[string]any{"backupId":backupID,"sizeBytes":stat.Size(),"expiresInSeconds":1800,"downloadPath":"/v1/backups/"+backupID})
}
func(s *Server)backupFile(w http.ResponseWriter,r *http.Request,id string){
 if len(id)!=48||strings.Trim(id,"0123456789abcdef")!=""{fail(w,404,"backup not found");return}
 s.mu.Lock()
 b,ok:=s.backups[id]
 if !ok||b.owner!=requestOwner(r)||s.now().After(b.expires){s.mu.Unlock();fail(w,404,"backup not found");return}
 if r.Method=="DELETE"{delete(s.backups,id)}
 s.mu.Unlock()
 if r.Method=="DELETE"{_=os.RemoveAll(filepath.Dir(b.path));w.WriteHeader(204);return}
 st,err:=os.Lstat(b.path)
 if err!=nil||!st.Mode().IsRegular()||st.Size()!=b.size{fail(w,404,"backup is unavailable");return}
 file,err:=os.Open(b.path)
 if err!=nil{fail(w,404,"backup is unavailable");return}
 defer file.Close()
 w.Header().Set("Content-Type","application/octet-stream")
 w.Header().Set("Content-Disposition",fmt.Sprintf("attachment; filename=\"advance-db-%s.sqlite\"",id[:12]))
 w.Header().Set("Cache-Control","no-store")
 w.Header().Set("X-Content-Type-Options","nosniff")
 http.ServeContent(w,r,"snapshot.sqlite",st.ModTime(),file)
}
