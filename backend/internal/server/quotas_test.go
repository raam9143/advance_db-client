package server
import (
 "strings"
 "testing"
 "time"
)
func TestConnectionsAreQuotaBoundPerOwner(t *testing.T){
 s:=New("secret");owner:="session-1"
 s.mu.Lock()
 for i:=0;i<6;i++{
  id:=strings.Repeat(string(rune('a'+i)),48)
  s.dbs[id]=&session{owner:owner,expiry:time.Now().Add(time.Hour)}
 }
 if s.checkQuotaLocked(owner){t.Error("allowed seventh session connection")}
 if !s.checkQuotaLocked("session-2"){t.Error("second tenant denied its own connection")}
 s.mu.Unlock()
}
