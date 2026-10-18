package egress
import "testing"
func TestExactDatabaseEndpointAllowlist(t *testing.T){
 t.Setenv("ADB_PRODUCTION","true")
 t.Setenv("DB_ENDPOINT_ALLOWLIST","db.internal:5432,127.0.0.1:3306,[::1]:15432")
 for _,tc:=range []struct{host string;port int;ok bool}{
  {"db.internal",5432,true},{"DB.INTERNAL",5432,true},{"db.internal",3306,false},
  {"db.internal.evil.example",5432,false},{"127.0.0.1",3306,true},
  {"::1",15432,true},{"169.254.169.254",80,false},{"db.internal",80,false},
 }{
  if got:=Allowed(tc.host,tc.port);got!=tc.ok{t.Errorf("%s:%d got=%t",tc.host,tc.port,got)}
 }
 t.Setenv("DB_ENDPOINT_ALLOWLIST","")
 if Allowed("db.internal",5432){t.Fatal("missing allowlist failed open")}
}
func TestRejectBroadAllowlist(t *testing.T){
 for _,s:=range []string{"*","*:*","example.com:*","0.0.0.0:0","169.254.169.254:80","db.example.com:80/path","db.example.com:80,","db.example.com:99999","[fe80::1]:3306","user@db:5432"}{
  if ValidateList(s)==nil{t.Errorf("accepted unsafe entry %q",s)}
 }
 if e:=ValidateList("postgres.example.net:5432,[2001:db8::10]:443");e!=nil{t.Fatal(e)}
}
