package redisclient
import(
 "bufio"
 "context"
 "io"
 "net"
 "strconv"
 "strings"
 "testing"
 "time"
)
func TestRejectEndpoints(t *testing.T){
 for _,c:=range []Config{{Host:"example.com",Port:6379,PlaintextLoopback:true},{Host:"localhost",Port:-1},{Host:"local\nnet",Port:6379},{Host:"localhost",Port:6379,Database:16}}{
  if _,e:=New(c);e==nil{t.Fatalf("accepted %+v",c)}
 }
}
func TestDenyUnsafeCommands(t *testing.T){
 c,_:=New(Config{Host:"localhost",Port:6389,PlaintextLoopback:true})
 for _,r:=range []Request{{Command:"SET",Key:"x"},{Command:"CONFIG"},{Command:"EVAL"},{Command:"FLUSHALL"},{Command:"LRANGE",Key:"a",Stop:300},{Command:"SCAN",Cursor:"-1"}}{
  if _,e:=c.Inspect(context.Background(),r);e==nil{t.Fatalf("accepted %+v",r)}
 }
}
func TestReadOnlyRESP2(t *testing.T){
 l,e:=net.Listen("tcp","127.0.0.1:0");if e!=nil{t.Fatal(e)};defer l.Close()
 errorsCh:=make(chan error,4)
 go func(){for i:=0;i<3;i++{
  conn,err:=l.Accept();if err!=nil{errorsCh<-err;return}
  go func(){defer conn.Close();rd:=bufio.NewReader(conn)
   for{
    line,err:=rd.ReadString('\n');if err!=nil{return}
    n,err:=strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line,"*")));if err!=nil{errorsCh<-err;return}
    args:=make([]string,n)
    for j:=range args{
     line,err=rd.ReadString('\n');if err!=nil{errorsCh<-err;return}
     size,er:=strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line,"$")));if er!=nil{errorsCh<-er;return}
     part:=make([]byte,size+2);if _,er=io.ReadFull(rd,part);er!=nil{errorsCh<-er;return}
     args[j]=string(part[:size])
    }
    switch args[0]{
    case "AUTH":_,_=conn.Write([]byte("+OK\r\n"))
    case "PING":_,_=conn.Write([]byte("+PONG\r\n"))
    case "GET":_,_=conn.Write([]byte("$5\r\nvalue\r\n"))
    case "SCAN":_,_=conn.Write([]byte("*2\r\n$1\r\n0\r\n*1\r\n$3\r\nkey\r\n"))
    default:_,_=conn.Write([]byte("-ERR forbidden\r\n"))
    }
   }
  }()
 }}()
 c,e:=New(Config{Host:"127.0.0.1",Port:l.Addr().(*net.TCPAddr).Port,Password:"secret",PlaintextLoopback:true});if e!=nil{t.Fatal(e)}
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
 if e=c.Ping(ctx);e!=nil{t.Fatal(e)}
 got,e:=c.Inspect(ctx,Request{Command:"GET",Key:"sample"});if e!=nil||got.Value!="value"{t.Fatalf("%+v %v",got,e)}
 got,e=c.Inspect(ctx,Request{Command:"SCAN"});if e!=nil||len(got.Value.([]any))!=2{t.Fatalf("%+v %v",got,e)}
 select{case e:=<-errorsCh:t.Fatal(e);default:}
}
func TestBoundedReply(t *testing.T){
 w:=&wire{rd:bufio.NewReader(strings.NewReader("$1048577\r\n")),budget:maxResponse};if _,e:=w.reply(0);e==nil{t.Fatal("oversize accepted")}
 w=&wire{rd:bufio.NewReader(strings.NewReader("*257\r\n")),budget:maxResponse};if _,e:=w.reply(0);e==nil{t.Fatal("huge array accepted")}
}
