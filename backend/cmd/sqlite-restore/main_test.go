package main
import(
 "database/sql"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func makeDB(t *testing.T,path string,value string){
 t.Helper()
 db,err:=sql.Open("sqlite",path);if err!=nil{t.Fatal(err)}
 if _,err=db.Exec("CREATE TABLE audit_sample (label TEXT NOT NULL)");err!=nil{t.Fatal(err)}
 if _,err=db.Exec("INSERT INTO audit_sample(label) VALUES (?)",value);err!=nil{t.Fatal(err)}
 if err=db.Close();err!=nil{t.Fatal(err)}
}
func labelAt(t *testing.T,path string)string{
 t.Helper();db,err:=sql.Open("sqlite","file:"+path+"?mode=ro");if err!=nil{t.Fatal(err)}
 defer db.Close()
 var s string;if err=db.QueryRow("SELECT label FROM audit_sample").Scan(&s);err!=nil{t.Fatal(err)}
 return s
}
func TestOfflineRestoreDryRunAndRollbackCopy(t *testing.T){
 root:=t.TempDir();backupDir:=filepath.Join(root,"backups");dataDir:=filepath.Join(root,"data")
 if err:=os.Mkdir(backupDir,0700);err!=nil{t.Fatal(err)}
 if err:=os.Mkdir(dataDir,0700);err!=nil{t.Fatal(err)}
 t.Setenv("SQLITE_ALLOWED_DIR",dataDir);t.Setenv("SQLITE_BACKUP_DIR",backupDir)
 dest:=filepath.Join(dataDir,"live.sqlite");source:=filepath.Join(backupDir,"snapshot.sqlite")
 makeDB(t,dest,"old");makeDB(t,source,"new")
 res,err:=restore(source,dest,false)
 if err!=nil||!strings.Contains(res,"dry run"){t.Fatalf("%s %v",res,err)}
 if got:=labelAt(t,dest);got!="old"{t.Fatalf("dry run modified DB: %s",got)}
 original,err:=restore(source,dest,true)
 if err!=nil{t.Fatal(err)}
 if got:=labelAt(t,dest);got!="new"{t.Fatalf("restore not applied: %s",got)}
 if got:=labelAt(t,original);got!="old"{t.Fatalf("original not retained: %s",got)}
 stat,err:=os.Stat(dest);if err!=nil||stat.Mode().Perm()!=0600{t.Fatal("restored file not private")}
}
func TestOfflineRestoreRejectsUnsafeFiles(t *testing.T){
 root:=t.TempDir();dbDir:=filepath.Join(root,"data");backupDir:=filepath.Join(root,"backups")
 _=os.Mkdir(dbDir,0700);_=os.Mkdir(backupDir,0700)
 t.Setenv("SQLITE_ALLOWED_DIR",dbDir);t.Setenv("SQLITE_BACKUP_DIR",backupDir)
 dest:=filepath.Join(dbDir,"live.sqlite");snapshot:=filepath.Join(backupDir,"snapshot.sqlite")
 makeDB(t,dest,"old");makeDB(t,snapshot,"new")
 if err:=os.WriteFile(dest+"-wal",[]byte("active"),0600);err!=nil{t.Fatal(err)}
 if _,err:=restore(snapshot,dest,true);err==nil{t.Fatal("restore over active WAL permitted")}
 _=os.Remove(dest+"-wal")
 if _,err:=restore(dest,dest,true);err==nil{t.Fatal("restore accepted target as source")}
 bad:=filepath.Join(backupDir,"broken.sqlite")
 _=os.WriteFile(bad,[]byte(strings.Repeat("x",1024)),0600)
 if _,err:=restore(bad,dest,true);err==nil{t.Fatal("corrupt SQLite accepted")}
 if got:=labelAt(t,dest);got!="old"{t.Fatal("failed restore modified the database")}
}
