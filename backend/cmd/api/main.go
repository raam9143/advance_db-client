package main

import (
	"github.com/raam9143/advance_db-client/backend/internal/server"
 "github.com/raam9143/advance_db-client/backend/internal/prodcheck"
 "github.com/raam9143/advance_db-client/backend/internal/vault"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
 if err:=prodcheck.ValidateEnv();err!=nil{log.Fatal(err)}
	address := os.Getenv("API_LISTEN")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	token := os.Getenv("API_TOKEN")
	if token == "" {
		log.Fatal("API_TOKEN must be set; refusing to start without API authentication")
	}
	h := server.New(token)
 if os.Getenv("ADB_PRODUCTION")=="true"{
  v,err:=vault.FromEnv()
  if err!=nil{log.Fatalf("production vault unavailable: %v",err)}
  if _,err=v.List();err!=nil{log.Fatalf("production vault inaccessible: %v",err)}
  if err=h.Ready();err!=nil{log.Fatal("production audit not ready")}
 }
	srv := &http.Server{Addr: address, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 35 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	log.Printf("database API listening on %s", address)
	log.Fatal(srv.ListenAndServe())
}
