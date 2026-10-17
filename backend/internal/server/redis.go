package server

import (
 "context"
 "crypto/rand"
 "encoding/hex"
 "net/http"
 "os"
 "strconv"
 "strings"
 "time"

 "github.com/raam9143/advance_db-client/backend/internal/redisclient"
)

// Redis sessions are explicitly read-only and never pass Redis input to SQL APIs.
func(s *Server) connectRedis(w http.ResponseWriter,r *http.Request,c ConnectionRequest){
 if strings.TrimSpace(c.Name)==""||len(c.Name)>120{fail(w,400,"connection name required");return}
 database:=0
 if c.Database!=""{
  n,err:=strconv.Atoi(c.Database);if err!=nil||n<0||n>15{fail(w,400,"Redis database must be between 0 and 15");return}
  database=n
 }
 mode:=c.SSLMode
 if mode==""{mode="require"}
 if mode!="require"&&mode!="verify-full"&&mode!="disable"{fail(w,400,"unsupported Redis TLS mode");return}
 plain:=mode=="disable"
 if plain&&os.Getenv("REDIS_ALLOW_PLAINTEXT_LOOPBACK")!="true"{fail(w,400,"plaintext Redis disabled by server policy");return}
 client,err:=redisclient.New(redisclient.Config{Host:c.Host,Port:c.Port,Username:c.User,Password:c.Password,Database:database,PlaintextLoopback:plain})
 if err!=nil{fail(w,400,err.Error());return}
 ctx,cancel:=context.WithTimeout(r.Context(),8*time.Second);defer cancel()
 if err:=client.Ping(ctx);err!=nil{fail(w,400,"Redis connection failed");return}
 var random [24]byte
 if _,err=rand.Read(random[:]);err!=nil{fail(w,500,"could not create connection");return}
 id:=hex.EncodeToString(random[:])
 info:=PublicConnection{ID:id,Name:c.Name,Engine:"redis",Database:strconv.Itoa(database),Host:c.Host,ReadOnly:true}
 s.mu.Lock()
 if !s.checkQuotaLocked(requestOwner(r)){s.mu.Unlock();fail(w,429,"connection limit reached");return}
 s.dbs[id]=&session{owner:requestOwner(r),info:info,redis:client,expiry:s.now().Add(connectionTTL)}
 s.mu.Unlock()
 respond(w,201,info)
}
func(s *Server) inspectRedis(w http.ResponseWriter,r *http.Request,id string){
 c,ok:=s.lookup(id,requestOwner(r))
 if !ok{fail(w,404,"connection not found");return}
 if c.redis==nil{fail(w,400,"not a Redis connection");return}
 var q redisclient.Request
 if !decode(w,r,&q){return}
 ctx,cancel:=context.WithTimeout(r.Context(),12*time.Second);defer cancel()
 result,err:=c.redis.Inspect(ctx,q)
 if err!=nil{fail(w,400,err.Error());return}
 respond(w,200,result)
}
