// Command adb-vault is a local operator-only credential vault provisioner.
package main

import(
 "encoding/json"
 "flag"
 "fmt"
 "io"
 "os"
 "sort"
 "github.com/raam9143/advance_db-client/backend/internal/vault"
)
func main(){
 if len(os.Args)<2{fmt.Fprintln(os.Stderr,"usage: adb-vault [list|put|rotate]");os.Exit(2)}
 store,err:=vault.FromEnv();if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 switch os.Args[1]{
 case "list":
  names,err:=store.List();if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
  sort.Strings(names)
  for _,name:=range names{fmt.Println(name)}
 case "put":
  opts:=flag.NewFlagSet("put",flag.ExitOnError)
  name:=opts.String("name","","credential reference")
  _=opts.Parse(os.Args[2:])
  if *name==""{fmt.Fprintln(os.Stderr,"-name is required");os.Exit(2)}
  // Read credentials from stdin; never accept a password via argv or log it.
  input,err:=io.ReadAll(io.LimitReader(os.Stdin,8193))
  if err!=nil||len(input)>8192{fmt.Fprintln(os.Stderr,"invalid credential input size");os.Exit(1)}
  var c vault.Credentials
  if err=json.Unmarshal(input,&c);err!=nil{fmt.Fprintln(os.Stderr,"invalid JSON");os.Exit(1)}
  if err=store.Put(*name,c);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
  fmt.Println("Encrypted credential stored:",*name)
 case "rotate":
  n,err:=store.Rotate()
  if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
  fmt.Printf("Re-encrypted %d credentials using active key\n",n)
 default:fmt.Fprintln(os.Stderr,"unknown command");os.Exit(2)
 }
}
