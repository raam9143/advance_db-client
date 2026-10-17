package server
import(
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "net/http"
)
type scopeKey struct{}
type roleKey struct{}
func requestRole(r *http.Request)string{role,_:=r.Context().Value(roleKey{}).(string);if role==""{return "admin"};return role}
func verifiedRole(token,id,role string)string{
 mac:=hmac.New(sha256.New,[]byte(token))
 mac.Write([]byte("adb-role:"+id+":"+role))
 return hex.EncodeToString(mac.Sum(nil))
}
func requestOwner(r *http.Request)string{
 owner,_:=r.Context().Value(scopeKey{}).(string)
 if owner==""{return "service"}
 return owner
}
// A signed UI namespace separates temporary DB connections for each browser login.
func (s *Server) owner(r *http.Request)(string,error){
 id:=r.Header.Get("X-ADB-Session-ID")
 sig:=r.Header.Get("X-ADB-Session-Proof")
 if id==""&&sig==""{return "service",nil}
 if len(id)!=64||len(sig)!=64{return "",errors.New("invalid session signature")}
 raw,err:=hex.DecodeString(id);if err!=nil||len(raw)!=sha256.Size{return "",errors.New("invalid session ID")}
 signature,err:=hex.DecodeString(sig);if err!=nil{return "",errors.New("invalid session proof")}
 mac:=hmac.New(sha256.New,[]byte(s.token))
 mac.Write([]byte("adb-session:"+id))
 if !hmac.Equal(signature,mac.Sum(nil)){return "",errors.New("scope signature mismatch")}
 role:=r.Header.Get("X-ADB-Role")
 proof:=r.Header.Get("X-ADB-Role-Proof")
 if role==""&&proof==""{
  // Legacy scoped requests may only browse data, never mutate data.
  role="viewer"
 }else{
  if role!="admin"&&role!="viewer"||len(proof)!=64{return "",errors.New("invalid scoped role")}
  signature,err:=hex.DecodeString(proof)
  if err!=nil||!hmac.Equal(signature,decodeRoleProof(s.token,id,role)){return "",errors.New("invalid signed scoped role")}
 }
 *r= *r.WithContext(context.WithValue(r.Context(),roleKey{},role))
 return id,nil
}
func decodeRoleProof(token,id,role string)[]byte{
 mac:=hmac.New(sha256.New,[]byte(token))
 mac.Write([]byte("adb-role:"+id+":"+role))
 return mac.Sum(nil)
}
func verifiedScope(token,id string)(string,string){
 mac:=hmac.New(sha256.New,[]byte(token))
 mac.Write([]byte("adb-session:"+id))
 return id,hex.EncodeToString(mac.Sum(nil))
}
