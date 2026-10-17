// Package mongoinspect provides read-only, bounded MongoDB collection inspection.
package mongoinspect

import (
 "context"
 "crypto/tls"
 "encoding/json"
 "errors"
 "net"
 "regexp"
 "sort"
 "strconv"
 "strings"
 "time"

 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
 "go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
 MaxDocumentBytes=1<<20
 MaxResultBytes=4<<20
 MaxLimit=100
 MaxOffset=10000
)
type Config struct {
 Host string
 Port int
 Database string
 Username string
 Password string
 PlaintextLoopback bool
}
type Client struct {
 db *mongo.Database
 client *mongo.Client
 name string
}
type FindRequest struct {
 Collection string `json:"collection"`
 Limit int `json:"limit"`
 Offset int `json:"offset"`
 FilterField string `json:"filterField,omitempty"`
 FilterValue json.RawMessage `json:"filterValue,omitempty"`
}
type FindResult struct {
 Collection string `json:"collection"`
 Documents []json.RawMessage `json:"documents"`
 Limit int `json:"limit"`
 Offset int `json:"offset"`
 NextOffset *int `json:"nextOffset"`
}
var validCollection=regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,119}$`)
var validField=regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]{0,119}$`)
func loopback(host string)bool{
 if strings.EqualFold(host,"localhost"){return true}
 ip:=net.ParseIP(host);return ip!=nil&&ip.IsLoopback()
}
func ValidateConfig(c Config)error{
 if c.Host==""||len(c.Host)>253||strings.ContainsAny(c.Host,"/\\\x00\r\n")||c.Port<1||c.Port>65535{return errors.New("invalid MongoDB endpoint")}
 if c.Database==""||len(c.Database)>63||strings.ContainsAny(c.Database,"/\\ .\"$\x00\r\n"){return errors.New("invalid MongoDB database name")}
 if len(c.Username)>256||len(c.Password)>1024{return errors.New("MongoDB credentials too long")}
 if c.PlaintextLoopback&&!loopback(c.Host){return errors.New("MongoDB plaintext only permitted on loopback")}
 if c.Username==""&&c.Password!=""{return errors.New("MongoDB username required with password")}
 return nil
}
func Dial(ctx context.Context,c Config)(*Client,error){
 if err:=ValidateConfig(c);err!=nil{return nil,err}
 cfg:=options.Client().SetHosts([]string{net.JoinHostPort(c.Host,strconv.Itoa(c.Port))}).
  SetDirect(true).SetServerSelectionTimeout(5*time.Second).SetConnectTimeout(5*time.Second).
  SetMaxPoolSize(3).SetMinPoolSize(0).SetRetryWrites(false)
 if !c.PlaintextLoopback{cfg.SetTLSConfig(&tls.Config{MinVersion:tls.VersionTLS12})}
 if c.Username!=""{cfg.SetAuth(options.Credential{Username:c.Username,Password:c.Password})}
 connection,err:=mongo.Connect(ctx,cfg)
 if err!=nil{return nil,errors.New("MongoDB connection failed")}
 if err=connection.Ping(ctx,readpref.Primary());err!=nil{
  cleanupCtx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
  _=connection.Disconnect(cleanupCtx)
  return nil,errors.New("MongoDB connection failed")
 }
 return &Client{db:connection.Database(c.Database),client:connection,name:c.Database},nil
}
func(c *Client)Close(){
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
 _=c.client.Disconnect(ctx)
}
func(c *Client)Collections(ctx context.Context)([]string,error){
 names,err:=c.db.ListCollectionNames(ctx,bson.D{})
 if err!=nil{return nil,errors.New("MongoDB collection listing failed")}
 if len(names)>1000{return nil,errors.New("collection count exceeds 1000")}
 sort.Strings(names)
 return names,nil
}
func ValidateFind(r FindRequest)(bson.D,error){
 if !validCollection.MatchString(r.Collection)||strings.HasPrefix(r.Collection,"system.")||strings.Contains(r.Collection,".."){return nil,errors.New("invalid collection")}
 if r.Limit<1||r.Limit>MaxLimit||r.Offset<0||r.Offset>MaxOffset{return nil,errors.New("pagination out of bounds")}
 if r.FilterField==""{
  if len(r.FilterValue)>0{return nil,errors.New("filter field required with filter value")}
  return bson.D{},nil
 }
 if !validField.MatchString(r.FilterField)||strings.Contains(r.FilterField,"..")||strings.HasPrefix(r.FilterField,"_")&&r.FilterField!="_id"{return nil,errors.New("invalid filter field")}
 if len(r.FilterValue)==0||len(r.FilterValue)>4096{return nil,errors.New("filter value must be a JSON scalar")}
 var value any
 if err:=json.Unmarshal(r.FilterValue,&value);err!=nil{return nil,errors.New("invalid filter JSON")}
 switch value.(type){case nil,bool,string,float64:default:return nil,errors.New("filter must be a JSON scalar")}
 if s,ok:=value.(string);ok&&len(s)>1024{return nil,errors.New("filter string too long")}
 return bson.D{{Key:r.FilterField,Value:value}},nil
}
func(c *Client)Find(ctx context.Context,r FindRequest)(FindResult,error){
 out:=FindResult{Collection:r.Collection,Limit:r.Limit,Offset:r.Offset,Documents:[]json.RawMessage{}}
 filter,err:=ValidateFind(r);if err!=nil{return out,err}
 opts:=options.Find().SetLimit(int64(r.Limit+1)).SetSkip(int64(r.Offset)).
  SetSort(bson.D{{Key:"_id",Value:1}}).SetBatchSize(int32(r.Limit+1)).
  SetMaxTime(8*time.Second)
 cursor,err:=c.db.Collection(r.Collection).Find(ctx,filter,opts)
 if err!=nil{return out,errors.New("MongoDB document query failed")}
 defer cursor.Close(ctx)
 size:=0
 for cursor.Next(ctx){
  if len(out.Documents)>=r.Limit{
   next:=r.Offset+r.Limit;out.NextOffset=&next;break
  }
  if len(cursor.Current)>MaxDocumentBytes{return out,errors.New("document exceeds 1 MiB limit")}
  doc,err:=bson.MarshalExtJSON(cursor.Current,true,false)
  if err!=nil{return out,errors.New("document serialization failed")}
  size+=len(doc)
  if size>MaxResultBytes{return out,errors.New("MongoDB result exceeds 4 MiB limit")}
  out.Documents=append(out.Documents,json.RawMessage(doc))
 }
 if err=cursor.Err();err!=nil{return out,errors.New("MongoDB document cursor failed")}
 return out,nil
}
