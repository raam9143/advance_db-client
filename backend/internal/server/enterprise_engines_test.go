package server
import(
 "net/url"
 "strings"
 "testing"
)
func TestEnterpriseDSN(t *testing.T){
 for _,tc:=range []struct{engine,driver string;port int}{
  {"sqlserver","sqlserver",1433},
  {"oracle","oracle",2484},
 }{
  driver,dsn,err:=ConfigDSN(ConnectionRequest{Name:"enterprise",Engine:tc.engine,Host:"db.example.com",Port:tc.port,Database:"APP",User:"alice",Password:"some@pass",SSLMode:"verify-full",ReadOnly:true})
  if err!=nil||driver!=tc.driver{t.Fatalf("%s driver=%s err=%v",tc.engine,driver,err)}
  parsed,err:=url.Parse(dsn)
  if err!=nil||parsed.User.Username()!="alice"{t.Fatalf("%s invalid URL: %v",tc.engine,err)}
  if !strings.Contains(dsn,"some")||!strings.Contains(dsn,"pass"){t.Fatalf("%s password is missing",tc.engine)}
  if tc.engine=="sqlserver"{
   if parsed.Query().Get("encrypt")!="true"||parsed.Query().Get("TrustServerCertificate")!="false"{t.Fatal("SQL Server certificate verification disabled")}
  }else{
   if !strings.Contains(strings.ToLower(dsn),"ssl"){t.Fatal("Oracle TCPS absent")}
  }
 }
 oracleSID:=ConnectionRequest{Name:"oracle sid",Engine:"oracle",Host:"oracle.example.com",Port:2484,Database:"ORCL",User:"u",Password:"p",ServiceMode:"sid"}
 _,dsn,err:=ConfigDSN(oracleSID)
 if err!=nil||!strings.Contains(strings.ToUpper(dsn),"SID"){t.Fatalf("SID not present: %v",err)}
}
func TestEnterpriseRejectInsecureTLS(t *testing.T){
 for _,engine:=range []string{"oracle","sqlserver"}{
  for _,mode:=range []string{"disable","false","verify-ca"}{
   _,_,err:=ConfigDSN(ConnectionRequest{Name:"x",Engine:engine,Host:"db.example.com",Port:1433,Database:"db",User:"user",Password:"secret",SSLMode:mode})
   if err==nil{t.Fatalf("%s allowed sslMode=%s",engine,mode)}
  }
 }
 _,_,err:=ConfigDSN(ConnectionRequest{Name:"x",Engine:"oracle",Host:"db.example.com",Port:2484,Database:"x",User:"u",Password:"p",ServiceMode:"invalid"})
 if err==nil{t.Fatal("accepted invalid Oracle serviceMode")}
}
func TestEnterprisePaging(t *testing.T){
 quoted,e:=quoteIdentifier("sqlserver","na]me");if e!=nil||quoted!="[na]]me]"{t.Fatalf("bad TDS identifier %q",quoted)}
 for _,engine:=range []string{"sqlserver","oracle","postgres"}{
  stmt,e:=tablePageSQL(engine,"sample",101,50)
  if e!=nil||!strings.Contains(stmt,"50")||!strings.Contains(stmt,"101"){t.Fatalf("%s: %q %v",engine,stmt,e)}
  if engine=="sqlserver"&&!strings.Contains(stmt,"ORDER BY"){t.Fatal("missing SQL Server ORDER BY")}
  if engine=="oracle"&&!strings.Contains(stmt,"FETCH NEXT"){t.Fatal("missing Oracle page clause")}
 }
}

// A SQL Server self-signed development certificate is allowed only on loopback,
// with a second opt-in server policy; external hosts never bypass validation.
func TestSQLServerLocalTLSOverride(t *testing.T){
 t.Setenv("SQLSERVER_ALLOW_UNVERIFIED_LOOPBACK","true")
 for _,host:=range []string{"db.example.com","192.168.3.14","8.8.8.8"}{
  _,_,e:=ConfigDSN(ConnectionRequest{Name:"test",Engine:"sqlserver",Host:host,Port:1433,Database:"master",User:"sa",Password:"pw",SSLMode:"local-insecure"})
  if e==nil{t.Fatalf("remote certificate verification bypassed for %s",host)}
 }
 _,dsn,e:=ConfigDSN(ConnectionRequest{Name:"test",Engine:"sqlserver",Host:"127.0.0.1",Port:1433,Database:"master",User:"sa",Password:"pw",SSLMode:"local-insecure"})
 if e!=nil{t.Fatal(e)}
 u,e:=url.Parse(dsn)
 if e!=nil||u.Query().Get("encrypt")!="true"||u.Query().Get("TrustServerCertificate")!="true"{t.Fatal("expected encrypted local-only test connection")}
 t.Setenv("SQLSERVER_ALLOW_UNVERIFIED_LOOPBACK","false")
 _,_,e=ConfigDSN(ConnectionRequest{Name:"test",Engine:"sqlserver",Host:"127.0.0.1",Port:1433,Database:"master",User:"sa",Password:"pw",SSLMode:"local-insecure"})
 if e==nil{t.Fatal("bypass accepted without explicit opt-in")}
}
