package server
import(
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "net/http"
)
type scopeKey struct{}
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
 return id,nil
}
func verifiedScope(token,id string)(string,string){
 mac:=hmac.New(sha256.New,[]byte(token))
 mac.Write([]byte("adb-session:"+id))
 return id,hex.EncodeToString(mac.Sum(nil))
}
