package server

import "strings"

// ReadOnlySQL is a conservative statement gate, not a SQL grammar or an
// authorization boundary. Use SQL-level read-only credentials in production.
// It rejects comments, dollar-quoted SQL, stacked statements, side-effect
// keywords/functions, and malformed quotation before a driver sees the input.
func ReadOnlySQL(input string) bool {
 if len(input)>100000{return false}
 tokens,ok:=readOnlyTokens(input)
 if !ok||len(tokens)==0{return false}
 first:=tokens[0]
 if first!="SELECT"&&first!="WITH"&&first!="EXPLAIN"&&first!="SHOW"{return false}
 if first=="SHOW"{
  if len(tokens)<2{return false}
  switch tokens[1]{
  case "TABLES","COLUMNS","INDEX","INDEXES","DATABASES": // narrow metadata-only subset
  default:return false
  }
 }
 if first=="EXPLAIN" {
  if len(tokens)<2||tokens[1]!="SELECT"&&tokens[1]!="WITH"{return false}
 }
 if first=="WITH"&&!containsWord(tokens,"SELECT"){return false}
 for i,t:=range tokens {
  if prohibitedWord(t){return false}
  if t=="FOR"&&i+1<len(tokens)&&tokens[i+1]=="UPDATE"{return false}
  if t=="LOCK"&&i+1<len(tokens){return false}
 }
 return true
}
func containsWord(tokens []string,word string)bool{
 for _,tok:=range tokens{if tok==word{return true}}
 return false
}
func prohibitedWord(token string)bool{
 switch token{
 case "INSERT","UPDATE","DELETE","DROP","CREATE","ALTER","TRUNCATE",
  "GRANT","REVOKE","COPY","INTO","OUTFILE","CALL","EXEC","EXECUTE",
  "MERGE","REPLACE","UPSERT","DO","SET","RESET","COMMIT","ROLLBACK",
  "SAVEPOINT","ATTACH","DETACH","VACUUM","ANALYZE","IMPORT","EXPORT",
  "LOAD_FILE","PG_READ_FILE","PG_SLEEP","SLEEP","BENCHMARK",
  "NEXTVAL","SETVAL","DBLINK_EXEC","PG_ADVISORY_LOCK","PG_ADVISORY_XACT_LOCK",
  "LO_EXPORT","SYS_EVAL","XP_CMDSHELL","UTL_HTTP","DBMS_SCHEDULER",
  "GET_LOCK","RELEASE_LOCK","IS_FREE_LOCK","NEXT","VALUE","FOR",
  "HANDLER","INSTALL","UNINSTALL","INFILE","BULK":return true
 }
 return strings.HasPrefix(token,"DBMS_")||strings.HasPrefix(token,"UTL_")||strings.HasPrefix(token,"XP_")
}
func readOnlyTokens(src string)([]string,bool){
 tokens:=make([]string,0,24)
 var token strings.Builder
 flush:=func(){if token.Len()>0{tokens=append(tokens,strings.ToUpper(token.String()));token.Reset()}}
 var quoted byte
 terminal:=false
 for i:=0;i<len(src);i++{
  c:=src[i]
  if quoted!=0 {
   if c==quoted||quoted==']'&&c==']'{
    if i+1<len(src)&&src[i+1]==c{i++;continue}
    quoted=0
   }
   if c==0{return nil,false}
   continue
  }
  if terminal {
   if c!=' '&&c!='\t'&&c!='\r'&&c!='\n'{return nil,false}
   continue
  }
  if c==0{return nil,false}
  if c=='-'&&i+1<len(src)&&src[i+1]=='-'||c=='/'&&i+1<len(src)&&src[i+1]=='*'||c=='*'&&i+1<len(src)&&src[i+1]=='/'||c=='#'{return nil,false}
  if c=='$'&&i+1<len(src)&&src[i+1]=='$'{return nil,false}
  if c=='$'&&i+1<len(src)&&(isLetter(src[i+1])||src[i+1]=='_'){return nil,false}
  if c=='\''||c=='"'||c=='`'||c=='['{
   flush()
   if c=='['{quoted=']'}else{quoted=c}
   continue
  }
  if c==';'{flush();terminal=true;continue}
  if isLetter(c)||c=='_'||c>='0'&&c<='9'{token.WriteByte(c);continue}
  flush()
 }
 flush()
 return tokens,quoted==0
}
func isLetter(c byte)bool{return c>='a'&&c<='z'||c>='A'&&c<='Z'}
