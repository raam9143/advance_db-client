package server
import "testing"

func TestReadOnlySQLLexicalBoundary(t *testing.T){
 for _,q:=range []string{
  "SELECT 'DELETE FROM people; -- a quoted string' AS note",
  "SELECT \"update\" FROM \"select\"",
  "SELECT [delete] FROM [audit events]",
  "SELECT `drop` FROM users",
  "SELECT 42;\n",
  "WITH vals AS (SELECT 42) SELECT * FROM vals",
  "EXPLAIN SELECT * FROM users",
  "SELECT * FROM customers WHERE name=$1",
 }{if !ReadOnlySQL(q){t.Errorf("safe SELECT rejected: %q",q)}}
 for _,q:=range []string{
  "SELECT 1; DROP TABLE x",
  "SELECT 1; --comment",
  "SELECT 1 /* comment */",
  "SELECT 1 # comment",
  "SELECT * FROM t INTO OUTFILE '/tmp/export'",
  "SELECT * INTO newtable FROM oldtable",
  "SELECT pg_sleep(100)",
  "SELECT nextval('seq')",
  "SELECT lo_export(123,'/tmp/secret')",
  "SELECT dblink_exec('x','delete from secrets')",
  "SELECT pg_advisory_lock(123)",
  "SELECT xp_cmdshell('ls')",
  "SELECT dbms_scheduler.create_job('job') FROM dual",
  "SELECT $$UPDATE t SET a=1$$",
  "SELECT $tag$ SELECT 1 $tag$",
  "WITH t AS (DELETE FROM secrets RETURNING *) SELECT * FROM t",
  "EXPLAIN ANALYZE SELECT 1",
  "SHOW VARIABLES",
  "SHOW GRANTS",
  "SELECT 'unclosed quote",
  "WITH d AS (SELECT 1) UPDATE users SET admin=1",
  "SELECT NEXT VALUE FOR seq",
 }{if ReadOnlySQL(q){t.Errorf("unsafe query admitted: %q",q)}}
}
