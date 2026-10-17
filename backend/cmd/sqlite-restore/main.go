// sqlite-restore is an offline, fail-safe SQLite snapshot validation and restore tool.
package main

import (
 "context"
 "crypto/rand"
 "database/sql"
 "encoding/hex"
 "errors"
 "flag"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "strings"
 "time"

 _ "modernc.org/sqlite"
)
const maxBytes=128<<20
func existingFile(path,root string)(string,error){
 if !filepath.IsAbs(path)||!filepath.IsAbs(root){return "",errors.New("paths must be absolute")}
 rootReal,err:=filepath.EvalSymlinks(root);if err!=nil{return "",err}
 pathReal,err:=filepath.EvalSymlinks(path);if err!=nil{return "",err}
 rel,err:=filepath.Rel(rootReal,pathReal)
 if err!=nil||rel=="."||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){return "",errors.New("file outside allowed directory")}
 stat,err:=os.Lstat(path);if err!=nil{return "",err}
 if !stat.Mode().IsRegular()||stat.Size()<100||stat.Size()>maxBytes{return "",errors.New("invalid size or file type")}
 return pathReal,nil
}
func checkSQLite(path string)error{
 db,err:=sql.Open("sqlite","file:"+path+"?mode=ro")
 if err!=nil{return err}
 defer db.Close()
 db.SetMaxOpenConns(1)
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 var check string
 if err=db.QueryRowContext(ctx,"PRAGMA quick_check").Scan(&check);err!=nil{return errors.New("SQLite integrity check failed")}
 if check!="ok"{return errors.New("SQLite integrity check returned failure")}
 return nil
}
func copySync(dst string,source string)error{
 in,err:=os.Open(source);if err!=nil{return err};defer in.Close()
 out,err:=os.OpenFile(dst,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if err!=nil{return err}
 _,err=io.Copy(out,io.LimitReader(in,maxBytes+1))
 if err==nil{err=out.Sync()}
 closeErr:=out.Close();if err==nil{err=closeErr}
 if err!=nil{_=os.Remove(dst)}
 return err
}
func restore(snapshot,database string,apply bool)(string,error){
 targetRoot:=os.Getenv("SQLITE_ALLOWED_DIR")
 backupRoot:=os.Getenv("SQLITE_BACKUP_DIR")
 if targetRoot==""||backupRoot==""{return "",errors.New("SQLITE_ALLOWED_DIR and SQLITE_BACKUP_DIR must be set")}
 dest,err:=existingFile(database,targetRoot);if err!=nil{return "",err}
 source,err:=existingFile(snapshot,backupRoot);if err!=nil{return "",err}
 if dest==source{return "",errors.New("source and database are identical")}
 for _,suffix:=range []string{"-wal","-shm","-journal"}{
  if _,err=os.Lstat(dest+suffix);err==nil{return "",errors.New("database sidecar detected; stop all database writers and checkpoint WAL before restore")}
  if !os.IsNotExist(err){return "",err}
 }
 if err=checkSQLite(dest);err!=nil{return "",fmt.Errorf("destination invalid: %w",err)}
 if err=checkSQLite(source);err!=nil{return "",fmt.Errorf("snapshot invalid: %w",err)}
 if !apply{return "Validated both databases; no files changed (dry run)",nil}
 dir:=filepath.Dir(dest)
 var token [12]byte
 if _,err=rand.Read(token[:]);err!=nil{return "",err}
 id:=hex.EncodeToString(token[:])
 stage:=filepath.Join(dir,".adb-restore-stage-"+id)
 backup:=filepath.Join(dir,".adb-before-restore-"+id+".sqlite")
 if err=copySync(stage,source);err!=nil{return "",err}
 defer os.Remove(stage)
 if err=checkSQLite(stage);err!=nil{return "",err}
 // Keep the old database intact until the stage has passed integrity checks.
 if err=os.Rename(dest,backup);err!=nil{return "",err}
 if err=os.Rename(stage,dest);err!=nil{
  rollback:=os.Rename(backup,dest)
  if rollback!=nil{return "",fmt.Errorf("restore failed and manual recovery needed at %s",backup)}
  return "",err
 }
 dirFile,err:=os.Open(dir);if err==nil{_=dirFile.Sync();_=dirFile.Close()}
 return backup,nil
}
func main(){
 snapshot:=flag.String("snapshot","","absolute path to private SQLite snapshot")
 database:=flag.String("database","","absolute path to offline target SQLite database")
 confirm:=flag.String("confirm","","enter RESTORE to execute; otherwise perform dry-run")
 flag.Parse()
 if *snapshot==""||*database==""{fmt.Fprintln(os.Stderr,"snapshot and database paths are required");os.Exit(2)}
 result,err:=restore(*snapshot,*database,*confirm=="RESTORE")
 if err!=nil{fmt.Fprintln(os.Stderr,"Restore refused:",err);os.Exit(1)}
 if *confirm=="RESTORE"{fmt.Println("Restore completed; retained original at:",result)}else{fmt.Println(result)}
}
