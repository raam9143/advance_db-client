package prodcheck
import(
 "encoding/base64"
 "strings"
 "testing"
)
func secureEnv()map[string]string{
 return map[string]string{
  "ADB_PRODUCTION":"true",
  "API_TOKEN":"abcdefghijklmnopqrstuvwxyz0123456789",
  "AUDIT_LOG_PATH":"/var/lib/adb/audit.log",
  "AUDIT_HMAC_KEY":"0123456789abcdefghijklmnopqrstuvwxyz",
  "AUDIT_SINK_URL":"https://audit.example.org/ingest",
  "AUDIT_SINK_TOKEN":"audit_sink_token_with_at_least_32_chars",
  "VAULT_FILE":"/var/lib/adb/vault/profiles.enc",
  "DB_ENDPOINT_ALLOWLIST":"db.example.com:5432",
  "VAULT_ACTIVE_KEY_B64":base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
 }
}
func validate(m map[string]string)error{return Validate(func(k string)string{return m[k]})}
func TestSecurityProfile(t *testing.T){
 if err:=validate(secureEnv());err!=nil{t.Fatal(err)}
 if err:=validate(map[string]string{});err!=nil{t.Fatal("development affected:",err)}
 for _,bad:=range []map[string]string{
  {"API_TOKEN":"abc"},{"AUDIT_HMAC_KEY":"weak"},{"VAULT_FILE":"vault.json"},
  {"VAULT_ACTIVE_KEY_B64":"incorrect"},{"ALLOW_WRITE_QUERIES":"true"},
  {"MONGO_ALLOW_PLAINTEXT_LOOPBACK":"true"},{"AI_ASSIST_ENABLED":"true"},
 }{
  env:=secureEnv();for k,v:=range bad{env[k]=v}
  if err:=validate(env);err==nil||!strings.Contains(err.Error(),"production profile"){t.Fatalf("accepted insecure profile: %v",bad)}
 }
}
