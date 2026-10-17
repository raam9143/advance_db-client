package server

import (
	"context"
	"crypto/rand"
 "crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
 "log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
 "github.com/raam9143/advance_db-client/backend/internal/redisclient"
 "github.com/raam9143/advance_db-client/backend/internal/sshtunnel"
 "github.com/raam9143/advance_db-client/backend/internal/mongoinspect"
 go_ora "github.com/sijms/go-ora/v2"
)

const (
	maxBody       = 1 << 20
	maxRows       = 1000
	connectionTTL = 2 * time.Hour
)

type ConnectionRequest struct {
	Name     string `json:"name"`
	Engine   string `json:"engine"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"sslMode"`
	ReadOnly bool   `json:"readOnly"`
 ServiceMode string `json:"serviceMode,omitempty"`
 Tunnel *sshtunnel.Config `json:"tunnel,omitempty"`
}
type PublicConnection struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Engine   string `json:"engine"`
	Database string `json:"database"`
	Host     string `json:"host"`
	ReadOnly bool   `json:"readOnly"`
}
type session struct {
	info   PublicConnection
	owner  string
	db     *sql.DB
 redis *redisclient.Client
 mongo *mongoinspect.Client
 tunnel *sshtunnel.Tunnel
	expiry time.Time
}
type Server struct {
	token string
	mu    sync.Mutex
	dbs   map[string]*session
 backups map[string]*backupSnapshot
	now   func() time.Time
 audit *auditLog
 auditError error
}

func New(token string) *Server {
	a,err:=openAuditLog()
 return &Server{token: token, dbs: map[string]*session{}, backups:map[string]*backupSnapshot{},now: time.Now,audit:a,auditError:err}
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path == "/health" && r.Method == "GET" {
		respond(w, 200, map[string]string{"status": "ok"})
		return
	}
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || s.token == "" || subtle.ConstantTimeCompare([]byte(auth), []byte(s.token)) != 1 {
		fail(w, 401, "unauthorized")
		return
	}
	owner, err := s.owner(r)
    if err != nil {fail(w,401,"invalid session scope");return}
    r=r.WithContext(context.WithValue(r.Context(),scopeKey{},owner))
    if s.auditError!=nil{fail(w,503,"audit log unavailable");return}
    if s.audit!=nil{
     if err:=s.audit.Append(r.Method,r.URL.Path,owner,"started",0);err!=nil{fail(w,503,"audit log write failed");return}
     aw:=&auditResponseWriter{ResponseWriter:w,status:200}
     defer func(){_=s.audit.Append(r.Method,r.URL.Path,owner,"finished",aw.status)}()
     w=aw
    }
    if r.ContentLength > maxBody {
		fail(w, 413, "request too large")
		return
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	}
	s.cleanup()
	switch {
	case strings.HasPrefix(r.URL.Path,"/v1/backups/")&&(r.Method=="GET"||r.Method=="DELETE"):
        s.backupFile(w,r,strings.TrimPrefix(r.URL.Path,"/v1/backups/"))
    case r.URL.Path == "/v1/capabilities" && r.Method == "GET":
        respond(w, 200, Capabilities())
    case r.URL.Path == "/v1/schema/diff" && r.Method == "POST":
        s.diffSchema(w,r)
    case r.URL.Path == "/v1/connections" && r.Method == "GET":
		s.list(w, r)
	case r.URL.Path == "/v1/connections" && r.Method == "POST":
		s.connect(w, r)
	case strings.HasPrefix(r.URL.Path, "/v1/connections/"):
		tail := strings.TrimPrefix(r.URL.Path, "/v1/connections/")
		parts := strings.Split(tail, "/")
		if len(parts)>0 && (len(parts[0])!=48||strings.Trim(parts[0],"0123456789abcdef")!=""){fail(w,404,"connection not found");return}
  if len(parts) == 1 && r.Method == "DELETE" {
			s.disconnect(w, parts[0], requestOwner(r))
			return
		}
		if len(parts)==2 && r.Method=="POST" && parts[1]=="backup" {
    s.createBackup(w,r,parts[0]);return
  }
  if len(parts)==3 && r.Method=="POST" && parts[1]=="mongo" && parts[2]=="find" {
   s.findMongo(w,r,parts[0]);return
  }
  if len(parts) == 2 && r.Method=="POST" && parts[1]=="redis" {
   s.inspectRedis(w,r,parts[0]);return
  }
  if len(parts)==2 {
   if c,found:=s.lookup(parts[0],requestOwner(r));found {
     if c.info.Engine=="redis"&&parts[1]!="schema"{fail(w,400,"Redis requires the read-only inspector endpoint");return}
     if c.info.Engine=="mongo"&&parts[1]!="schema"{fail(w,400,"MongoDB requires the read-only document endpoint");return}
   }
  }
  if len(parts) == 2 && r.Method == "GET" && parts[1] == "schema" {
			s.schema(w, r, parts[0])
			return
		}
  if len(parts)==2 && r.Method=="GET" && parts[1]=="describe"{s.describe(w,r,parts[0]);return}
  if len(parts)==2 && r.Method=="GET" && parts[1]=="table"{s.tablePage(w,r,parts[0]);return}
		if len(parts) == 2 && r.Method == "POST" && parts[1] == "query" {
			s.query(w, r, parts[0])
			return
		}
		if len(parts) == 2 && r.Method == "POST" && parts[1] == "explain" {
        s.explain(w, r, parts[0]); return
    }
    if len(parts) == 2 && r.Method == "POST" && parts[1] == "export" {
			s.export(w, r, parts[0])
			return
		}
		fail(w, 404, "route not found")
	default:
		fail(w, 404, "route not found")
	}
}
func (s *Server) cleanup() {
 s.mu.Lock()
	defer s.mu.Unlock()
	for id,b:=range s.backups{
  if s.now().After(b.expires){delete(s.backups,id);_ = os.RemoveAll(filepath.Dir(b.path))}
 }
 for id, c := range s.dbs {
  if s.now().After(c.expiry) {
			if c.db!=nil{_ = c.db.Close()}
   if c.mongo!=nil{c.mongo.Close()}
   if c.tunnel!=nil{c.tunnel.Close()}
   delete(s.dbs, id)
		}
	}
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := make([]PublicConnection, 0, len(s.dbs))
	for _, c := range s.dbs {
		if c.owner==requestOwner(r){a=append(a,c.info)}
	}
	respond(w, 200, a)
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, 400, "invalid JSON payload")
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, 400, "unexpected JSON content")
		return false
	}
	return true
}
func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	var c ConnectionRequest
	if !decode(w, r, &c) {
		return
	}
	c.Engine = strings.ToLower(strings.TrimSpace(c.Engine))
 if c.Engine=="redis"{s.connectRedis(w,r,c);return}
 if c.Engine=="mongo"{s.connectMongo(w,r,c);return}
 originalHost:=c.Host
 var tunnel *sshtunnel.Tunnel
 if c.Tunnel!=nil{
  if c.Engine!="postgres"&&c.Engine!="cockroach"&&c.Engine!="mysql"&&c.Engine!="mariadb"{
   fail(w,400,"SSH tunneling currently supports PostgreSQL and MySQL protocols only");return
  }
  if (c.Engine=="postgres"||c.Engine=="cockroach")&&c.SSLMode!="disable"||
     (c.Engine=="mysql"||c.Engine=="mariadb")&&c.SSLMode!="false"{
    fail(w,400,"database TLS must be explicitly disabled for an SSH-local loopback target");return
  }
  ctx,cancel:=context.WithTimeout(r.Context(),12*time.Second)
  defer cancel()
  var host string
  var port int
  var err error
  tunnel,host,port,err=sshtunnel.Open(ctx,*c.Tunnel,c.Host,c.Port)
  if err!=nil{fail(w,400,err.Error());return}
  c.Host=host;c.Port=port
 }
 driver, dsn, err := ConfigDSN(c)
	if err != nil {
  if tunnel!=nil{tunnel.Close()}
  fail(w, 400, err.Error())
  return
 }
 db, err := sql.Open(driver, dsn)
	if err != nil {
  if tunnel!=nil{tunnel.Close()}
  fail(w, 400, "unable to create driver")
  return
 }
	if c.Engine == "sqlite" || c.Engine == "clickhouse" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(3)
	}
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(25 * time.Minute)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
  if tunnel!=nil{tunnel.Close()}
		fail(w, 400, "database connection failed")
		return
	}
	if c.Engine == "sqlite" && c.ReadOnly && c.Database == ":memory:" {
		if _, err = db.ExecContext(ctx, "PRAGMA query_only = ON"); err != nil {
			db.Close()
  if tunnel!=nil{tunnel.Close()}
			fail(w, 400, "could not enable SQLite read-only mode")
			return
		}
	}
	var b [24]byte
	if _, err = rand.Read(b[:]); err != nil {
		db.Close()
  if tunnel!=nil{tunnel.Close()}
		fail(w, 500, "could not create connection id")
		return
	}
	id := hex.EncodeToString(b[:])
	if c.Engine=="clickhouse"{c.ReadOnly=true}
    info := PublicConnection{ID: id, Name: c.Name, Engine: c.Engine, Database: c.Database, Host: originalHost, ReadOnly: c.ReadOnly}
	s.mu.Lock()
	if len(s.dbs)>=24{s.mu.Unlock();db.Close();if tunnel!=nil{tunnel.Close()};fail(w,429,"connection limit reached");return}
  s.dbs[id] = &session{info: info, owner: requestOwner(r), db: db,tunnel:tunnel, expiry: s.now().Add(connectionTTL)}
	s.mu.Unlock()
	respond(w, 201, info)
}
func (s *Server) disconnect(w http.ResponseWriter, id string, owner string) {
	s.mu.Lock()
	c, ok := s.dbs[id]
	if ok && c.owner==owner {delete(s.dbs,id)} else {ok=false}
	s.mu.Unlock()
	if !ok {
		fail(w, 404, "connection not found")
		return
	}
	if c.db!=nil{_ = c.db.Close()}
 if c.mongo!=nil{c.mongo.Close()}
 if c.tunnel!=nil{c.tunnel.Close()}
 w.WriteHeader(204)
}
func (s *Server) lookup(id, owner string) (*session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.dbs[id]
	if ok && c.owner != owner {return nil,false}
    if ok {
        c.expiry = s.now().Add(connectionTTL)
	}
	return c, ok
}
func (s *Server) schema(w http.ResponseWriter, r *http.Request, id string) {
	c, ok := s.lookup(id, requestOwner(r))
	if !ok {
		fail(w, 404, "connection not found")
		return
	}
	if c.info.Engine=="redis"{respond(w,200,[]map[string]string{});return}
 if c.info.Engine=="mongo"{s.mongoCollections(w,r,c);return}
 q, err := schemaSQL(c.info.Engine)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		fail(w, 400, "could not load schema")
		return
	}
	defer rows.Close()
	var tables []map[string]string
	for rows.Next() {
		var schema, name string
		if err = rows.Scan(&schema, &name); err != nil {
			fail(w, 500, "schema scan failed")
			return
		}
		tables = append(tables, map[string]string{"schema": schema, "name": name})
	}
	if err = rows.Err(); err != nil {
		fail(w, 500, "schema read failed")
		return
	}
	if tables == nil {
		tables = []map[string]string{}
	}
	respond(w, 200, tables)
}

type QueryRequest struct {
	SQL            string `json:"sql"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
    Parameters []any `json:"parameters,omitempty"`
    ConfirmWrite bool `json:"confirmWrite,omitempty"`
}
type QueryResult struct {
	Columns    []string `json:"columns"`
	Rows       [][]any  `json:"rows"`
	RowCount   int      `json:"rowCount"`
	Truncated  bool     `json:"truncated"`
	DurationMS int64    `json:"durationMs"`
	Affected   int64    `json:"affected,omitempty"`
}

func (s *Server) run(ctx context.Context, c *session, q QueryRequest) (QueryResult, error) {
	var out QueryResult
 if c.db==nil{return out,errors.New("SQL is not supported by this connection")}
 out.Columns = []string{}
	out.Rows = [][]any{}
	if strings.TrimSpace(q.SQL) == "" || len(q.SQL) > 100000 {
		return out, errors.New("SQL is required and must be under 100KB")
	}
	if q.TimeoutSeconds < 1 {
		q.TimeoutSeconds = 15
	}
	if q.TimeoutSeconds > 30 {
		return out, errors.New("query timeout cannot exceed 30 seconds")
	}
	args, err := ValidateParameters(q.Parameters)
    if err != nil { return out, err }
    if c.info.ReadOnly && !ReadOnlySQL(q.SQL) {
		return out, errors.New("connection is read-only; only a single SELECT/WITH/EXPLAIN statement is allowed")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(q.TimeoutSeconds)*time.Second)
	defer cancel()
	started := time.Now()
	if !ReadOnlySQL(q.SQL) {
		if !q.ConfirmWrite{return out,errors.New("write query requires explicit confirmation")}
        if os.Getenv("ALLOW_WRITE_QUERIES")!="true"{return out,errors.New("write queries are disabled by server policy")}
  if c.info.ReadOnly{return out,errors.New("read-only connection cannot execute writes")}
  result, err := c.db.ExecContext(ctx, q.SQL, args...)
  fingerprint:=sha256.Sum256([]byte(q.SQL))
  log.Printf("db_write engine=%s connection_id=%s statement_sha256=%x success=%t",c.info.Engine,c.info.ID,fingerprint,err==nil)
		if err != nil {
			return out, errors.New("SQL execution failed")
		}
		out.Affected, _ = result.RowsAffected()
		out.DurationMS = time.Since(started).Milliseconds()
		return out, nil
	}
	var rows *sql.Rows
 if c.info.Engine=="sqlserver"||c.info.Engine=="oracle" {
  // Vendor drivers may not support read-only transactions; use DB-level read-only roles.
  rows,err=c.db.QueryContext(ctx,q.SQL,args...)
 }else{
  var tx *sql.Tx
  tx,err=c.db.BeginTx(ctx,&sql.TxOptions{ReadOnly:true})
  if err!=nil{return out,errors.New("could not begin read-only transaction")}
  defer tx.Rollback()
  rows,err=tx.QueryContext(ctx,q.SQL,args...)
 }
	if err != nil {
		return out, errors.New("SQL query failed")
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return out, errors.New("could not read result columns")
	}
	out.Columns = cols
	for rows.Next() {
		if out.RowCount == maxRows {
			out.Truncated = true
			break
		}
		raw := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range raw {
			ptr[i] = &raw[i]
		}
		if err = rows.Scan(ptr...); err != nil {
			return out, errors.New("could not scan query result")
		}
		for i, v := range raw {
			if b, ok := v.([]byte); ok {
				raw[i] = string(b)
			}
		}
		out.Rows = append(out.Rows, raw)
		out.RowCount++
	}
	if err = rows.Err(); err != nil {
		return out, errors.New("result read failed")
	}
	out.DurationMS = time.Since(started).Milliseconds()
	return out, nil
}
func (s *Server) query(w http.ResponseWriter, r *http.Request, id string) {
	c, ok := s.lookup(id, requestOwner(r))
	if !ok {
		fail(w, 404, "connection not found")
		return
	}
	var q QueryRequest
	if !decode(w, r, &q) {
		return
	}
	out, err := s.run(r.Context(), c, q)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	respond(w, 200, out)
}
func (s *Server) export(w http.ResponseWriter, r *http.Request, id string) {
	c, ok := s.lookup(id, requestOwner(r))
	if !ok {
		fail(w, 404, "connection not found")
		return
	}
	var q QueryRequest
	if !decode(w, r, &q) {
		return
	}
	if !ReadOnlySQL(q.SQL) {
		fail(w, 400, "CSV export requires a single read-only query")
		return
	}
	result, err := s.run(r.Context(), c, q)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	format := r.URL.Query().Get("format")
    if format=="json"{w.Header().Set("Content-Disposition","attachment; filename=results.json");respond(w,200,result);return}
    if format!=""&&format!="csv"{fail(w,400,"unsupported export format");return}
    w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=results.csv")
	w.WriteHeader(200)
	writer := csv.NewWriter(w)
	_ = writer.Write(result.Columns)
	for _, row := range result.Rows {
		r := make([]string, len(row))
		for i, x := range row {
			if x != nil {
				r[i] = fmt.Sprint(x)
			}
		}
		for i:=range r{r[i]=safeCSVCell(r[i])}
  _ = writer.Write(r)
	}
	writer.Flush()
}
func safeCSVCell(value string)string{
 trimmed:=strings.TrimLeft(value," \t\r\n")
 if trimmed!=""&&strings.ContainsRune("=+-@",rune(trimmed[0])){return "'"+value}
 return value
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}
func schemaSQL(engine string) (string, error) {
	switch strings.ToLower(engine) {
	case "sqlserver":
  return "SELECT s.name,t.name FROM sys.tables t JOIN sys.schemas s ON t.schema_id=s.schema_id WHERE t.is_ms_shipped=0 ORDER BY s.name,t.name",nil
 case "oracle":
  return "SELECT owner,table_name FROM all_tables WHERE owner=USER ORDER BY owner,table_name",nil
 case "postgres","cockroach":
  return "SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') AND table_type='BASE TABLE' ORDER BY 1,2", nil
	case "mysql","mariadb":
		return "SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type='BASE TABLE' ORDER BY 1,2", nil
	case "clickhouse":
        return "SELECT database,name FROM system.tables WHERE database=currentDatabase() AND is_temporary=0 ORDER BY database,name",nil
    case "sqlite":
		return "SELECT 'main', name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name", nil
	default:
		return "", errors.New("unsupported engine")
	}
}
func ConfigDSN(c ConnectionRequest) (driver, dsn string, err error) {
	engine := strings.ToLower(strings.TrimSpace(c.Engine))
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 120 {
		return "", "", errors.New("connection name required (max 120 characters)")
	}
	if strings.ContainsAny(c.Host, "\x00\n\r") || strings.ContainsAny(c.Database, "\x00\n\r") {
		return "", "", errors.New("invalid connection fields")
	}
	if engine == "sqlite" {
		if c.Database == ":memory:" {
			return "sqlite", ":memory:", nil
		}
		if strings.HasPrefix(c.Database, "file:") {
			return "", "", errors.New("SQLite URI parameters are not allowed")
		}
		root := os.Getenv("SQLITE_ALLOWED_DIR")
		if root == "" {
			return "", "", errors.New("SQLITE_ALLOWED_DIR must be configured for SQLite files")
		}
		rootReal, e := filepath.EvalSymlinks(root)
		if e != nil {
			return "", "", errors.New("SQLite allowed directory does not exist")
		}
		path := c.Database
		if !filepath.IsAbs(path) {
			path = filepath.Join(rootReal, path)
		}
		pathReal, e := filepath.EvalSymlinks(path)
		if e != nil {
			return "", "", errors.New("SQLite file must already exist")
		}
		rel, e := filepath.Rel(rootReal, pathReal)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", "", errors.New("SQLite file must be within SQLITE_ALLOWED_DIR")
		}
		dsn := "file:" + pathReal
		if c.ReadOnly {
			dsn += "?mode=ro"
		}
		return "sqlite", dsn, nil
	}
	if c.Host == "" || c.Database == "" || c.User == "" {
		return "", "", errors.New("host, database and user required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return "", "", errors.New("invalid database port")
	}
switch engine {
 case "sqlserver":
  if c.ServiceMode!=""{return "","",errors.New("serviceMode is Oracle-only")}
  mode:=c.SSLMode;if mode==""{mode="verify-full"}
  localDev:=mode=="local-insecure" && os.Getenv("SQLSERVER_ALLOW_UNVERIFIED_LOOPBACK")=="true" && (strings.EqualFold(c.Host,"localhost")||func()bool{ip:=net.ParseIP(c.Host);return ip!=nil&&ip.IsLoopback()}())
  if mode!="require"&&mode!="verify-full"&&!localDev{return "","",errors.New("SQL Server requires verified TLS (or explicit local-only test mode)")}
  u:=&url.URL{Scheme:"sqlserver",Host:net.JoinHostPort(c.Host,fmt.Sprint(c.Port)),User:url.UserPassword(c.User,c.Password)}
  values:=url.Values{};values.Set("database",c.Database);values.Set("encrypt","true");values.Set("TrustServerCertificate",fmt.Sprint(localDev));values.Set("dial timeout","8")
  u.RawQuery=values.Encode()
  return "sqlserver",u.String(),nil
 case "oracle":
  mode:=c.SSLMode;if mode==""{mode="verify-full"}
  if mode!="require"&&mode!="verify-full"{return "","",errors.New("Oracle requires certificate-verified TCPS")}
  if c.ServiceMode!=""&&c.ServiceMode!="service"&&c.ServiceMode!="sid"{return "","",errors.New("invalid Oracle serviceMode")}
  options:=map[string]string{"SSL":"true","SSL VERIFY":"true","CONNECTION TIMEOUT":"8"}
  service:=c.Database
  if c.ServiceMode=="sid"{options["SID"]=c.Database;service=""}
  return "oracle",go_ora.BuildUrl(c.Host,c.Port,service,c.User,c.Password,options),nil
 case "clickhouse":
		if c.SSLMode!="" && c.SSLMode!="require" && c.SSLMode!="verify-full" {return "","",errors.New("ClickHouse requires verified TLS")}
		u:=&url.URL{Scheme:"https",Host:net.JoinHostPort(c.Host,fmt.Sprint(c.Port)),Path:"/",User:url.UserPassword(c.User,c.Password)}
		params:=url.Values{};params.Set("database",c.Database);u.RawQuery=params.Encode()
		return "adbclickhouse",u.String(),nil
	case "postgres","cockroach":
		mode := c.SSLMode
		if mode == "" {
			mode = "require"
		}
		if mode != "require" && mode != "verify-full" && mode != "verify-ca" && mode != "disable" {
			return "", "", errors.New("unsupported Postgres SSL mode")
		}
		u := &url.URL{Scheme: "postgres", User: url.UserPassword(c.User, c.Password), Host: net.JoinHostPort(c.Host, fmt.Sprint(c.Port)), Path: "/" + c.Database}
		values := url.Values{}
		values.Set("sslmode", mode)
		u.RawQuery = values.Encode()
		return "pgx", u.String(), nil
	case "mysql","mariadb":
		// MySQL driver uses parseTime and TLS when supported. TLS custom cert deployment requires a custom driver registration.
		if c.SSLMode != "" && c.SSLMode != "true" && c.SSLMode != "false" {
			return "", "", errors.New("MySQL sslMode must be true or false")
		}
		tls := "true"
		if c.SSLMode == "false" {
			tls = "false"
		}
		if strings.ContainsAny(c.User, "@/\x00") || strings.ContainsAny(c.Password, "\x00") || strings.ContainsAny(c.Database, "?\x00") {
			return "", "", errors.New("unsupported MySQL DSN field characters")
		}
		return "mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&tls=%s&timeout=8s&readTimeout=30s&writeTimeout=30s", c.User, c.Password, net.JoinHostPort(c.Host, fmt.Sprint(c.Port)), url.PathEscape(c.Database), tls), nil
	default:
		return "", "", errors.New("supported engines: postgres, cockroach, mysql, mariadb, sqlite, clickhouse, sqlserver, oracle")
	}
}

// ReadOnlySQL implements an intentionally conservative guard, not a full SQL parser. Database read-only roles are still required.
func ReadOnlySQL(input string) bool {
	s := strings.TrimSpace(input)
	if s == "" {
		return false
	}
	// Reject semicolons except a single terminator, line/block comments, and suspicious write keywords.
	if strings.Contains(s, "--") || strings.Contains(s, "/*") || strings.Contains(s, "*/") {
		return false
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, ";"))
	if strings.Contains(s, ";") {
		return false
	}
	u := strings.ToUpper(s)
	if !(strings.HasPrefix(u, "SELECT ") || strings.HasPrefix(u, "WITH ") || strings.HasPrefix(u, "EXPLAIN ") || strings.HasPrefix(u, "SHOW ")) {
		return false
	}
	for _, k := range []string{" INSERT ", " UPDATE ", " DELETE ", " DROP ", " CREATE ", " ALTER ", " TRUNCATE ", " GRANT ", " REVOKE ", " COPY ", " INTO OUTFILE", " INTO ", " NEXT VALUE FOR ", " FOR UPDATE", " PG_SLEEP(", " LOAD_FILE(", " DML ", " CALL ", " EXEC ", " EXECUTE "} {
		if strings.Contains(" "+u+" ", k) {
			return false
		}
	}
	return true
}
