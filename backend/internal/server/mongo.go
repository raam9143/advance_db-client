package server

import(
 "context"
 "crypto/rand"
 "encoding/hex"
 "net"
 "net/http"
 "os"
 "strings"
 "time"

 "github.com/raam9143/advance_db-client/backend/internal/mongoinspect"
)
func(s *Server)connectMongo(w http.ResponseWriter,r *http.Request,c ConnectionRequest){
 if strings.TrimSpace(c.Name)==""||len(c.Name)>120{fail(w,400,"connection name required");return}
 mode:=c.SSLMode
 if mode==""{mode="verify-full"}
 localOnly:=mode=="disable" && os.Getenv("MONGO_ALLOW_PLAINTEXT_LOOPBACK")=="true" && (strings.EqualFold(c.Host,"localhost")||func()bool{ip:=net.ParseIP(c.Host);return ip!=nil&&ip.IsLoopback()}())
 if mode!="require"&&mode!="verify-full"&&!localOnly{fail(w,400,"MongoDB requires verified TLS (or explicitly permitted loopback-only development mode)");return}
 cfg:=mongoinspect.Config{Host:c.Host,Port:c.Port,Database:c.Database,Username:c.User,Password:c.Password,PlaintextLoopback:localOnly}
 ctx,cancel:=context.WithTimeout(r.Context(),8*time.Second);defer cancel()
 client,err:=mongoinspect.Dial(ctx,cfg)
 if err!=nil{fail(w,400,err.Error());return}
 var token [24]byte
 if _,err=rand.Read(token[:]);err!=nil{client.Close();fail(w,500,"connection ID generation failed");return}
 id:=hex.EncodeToString(token[:])
 info:=PublicConnection{ID:id,Name:c.Name,Engine:"mongo",Database:c.Database,Host:c.Host,ReadOnly:true}
 s.mu.Lock()
 if len(s.dbs)>=24{s.mu.Unlock();client.Close();fail(w,429,"connection limit reached");return}
 s.dbs[id]=&session{owner:requestOwner(r),info:info,mongo:client,expiry:s.now().Add(connectionTTL)}
 s.mu.Unlock()
 respond(w,201,info)
}
func(s *Server)mongoCollections(w http.ResponseWriter,r *http.Request,c *session){
 ctx,cancel:=context.WithTimeout(r.Context(),12*time.Second);defer cancel()
 names,err:=c.mongo.Collections(ctx)
 if err!=nil{fail(w,400,err.Error());return}
 tables:=make([]map[string]string,0,len(names))
 for _,name:=range names{tables=append(tables,map[string]string{"schema":c.info.Database,"name":name})}
 respond(w,200,tables)
}
func(s *Server)findMongo(w http.ResponseWriter,r *http.Request,id string){
 c,found:=s.lookup(id,requestOwner(r))
 if !found{fail(w,404,"connection not found");return}
 if c.mongo==nil{fail(w,400,"not a MongoDB connection");return}
 var q mongoinspect.FindRequest
 if !decode(w,r,&q){return}
 ctx,cancel:=context.WithTimeout(r.Context(),12*time.Second);defer cancel()
 result,err:=c.mongo.Find(ctx,q)
 if err!=nil{fail(w,400,err.Error());return}
 respond(w,200,result)
}
