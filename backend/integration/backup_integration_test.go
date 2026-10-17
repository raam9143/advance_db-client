package integration
import(
 "bytes"
 "encoding/json"
 "os"
 "strings"
 "testing"
 "github.com/raam9143/advance_db-client/backend/internal/server"
)
func TestSQLiteSnapshotDownload(t *testing.T){
 root:=t.TempDir()
 if err:=os.Chmod(root,0700);err!=nil{t.Fatal(err)}
 t.Setenv("SQLITE_BACKUP_DIR",root)
 h:=server.New("integration-test")
 status,body:=call(t,h,"POST","/v1/connections",server.ConnectionRequest{Name:"snapshot",Engine:"sqlite",Database:":memory:"})
 check(t,status,body,201)
 var c server.PublicConnection
 if e:=json.Unmarshal(body,&c);e!=nil{t.Fatal(e)}
 path:="/v1/connections/"+c.ID+"/backup"
 status,body=call(t,h,"POST",path,map[string]any{"confirmBackup":false});check(t,status,body,400)
 status,body=call(t,h,"POST",path,map[string]any{"confirmBackup":true});check(t,status,body,201)
 var snap struct{BackupID string `json:"backupId"`;Size int64 `json:"sizeBytes"`}
 if e:=json.Unmarshal(body,&snap);e!=nil{t.Fatal(e)}
 if len(snap.BackupID)!=48||snap.Size<100{t.Fatalf("invalid snapshot %+v",snap)}
 download:="/v1/backups/"+snap.BackupID
 status,body=call(t,h,"GET",download,nil);check(t,status,body,200)
 if !bytes.HasPrefix(body,[]byte("SQLite format 3")){t.Fatalf("invalid SQLite snapshot header %q",body[:min(len(body),30)])}
 status,body=call(t,h,"DELETE",download,nil);check(t,status,body,204)
 status,body=call(t,h,"GET",download,nil);check(t,status,body,404)
 entries,err:=os.ReadDir(root);if err!=nil{t.Fatal(err)}
 if len(entries)!=0{t.Fatal("snapshot directory not removed")}
}
func TestSQLiteSnapshotPrivateRoot(t *testing.T){
 root:=t.TempDir()
 if e:=os.Chmod(root,0755);e!=nil{t.Fatal(e)}
 t.Setenv("SQLITE_BACKUP_DIR",root)
 h:=server.New("integration-test")
 status,body:=call(t,h,"POST","/v1/connections",server.ConnectionRequest{Name:"snapshot",Engine:"sqlite",Database:":memory:"})
 check(t,status,body,201)
 var c server.PublicConnection
 if e:=json.Unmarshal(body,&c);e!=nil{t.Fatal(e)}
 status,body=call(t,h,"POST","/v1/connections/"+c.ID+"/backup",map[string]any{"confirmBackup":true})
 check(t,status,body,503)
 if !strings.Contains(string(body),"permissions"){t.Fatalf("unexpected denial %s",body)}
}
