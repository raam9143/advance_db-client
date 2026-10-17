package server
import ("net/url";"testing")
func TestQuoteIdentifier(t *testing.T){
 for _,tc:=range []struct{engine,name,want string}{{"postgres","simple","\"simple\""},{"postgres","a\"b","\"a\"\"b\""},{"mysql","a"+string(rune(96))+"b",string(rune(96))+"a"+string([]rune{96,96})+"b"+string(rune(96))}}{
  got,e:=quoteIdentifier(tc.engine,tc.name);if e!=nil||got!=tc.want{t.Errorf("%v got=%q err=%v",tc,got,e)}
 }
 for _,bad:=range []string{"","\n","ab\x00c"}{if _,e:=quoteIdentifier("postgres",bad);e==nil{t.Errorf("accepted %q",bad)}}
}
func TestQualifiedName(t *testing.T){if _,e:=qualifiedName("sqlite","temp","tbl");e==nil{t.Fatal("attached schema accepted")};if _,e:=qualifiedName("mysql","","tbl");e==nil{t.Fatal("empty schema accepted")}}
func TestParsePage(t *testing.T){
 for _,tc:=range []struct{raw string;okay bool}{{"",true},{"limit=250&offset=15",true},{"limit=0",false},{"limit=251",false},{"offset=-1",false},{"offset=10000001",false},{"limit=oops",false}}{
 q,_:=url.ParseQuery(tc.raw);_,_,e:=parsePage(q);if (e==nil)!=tc.okay{t.Errorf("%q: %v",tc.raw,e)}
 }
}
