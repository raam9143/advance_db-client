package server

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "testing"
)

func TestAuditReceiptReplaysOnlyUnacknowledgedEvents(t *testing.T) {
    root := t.TempDir()
    if err := os.Chmod(root,0700); err != nil { t.Fatal(err) }
    t.Setenv("AUDIT_LOG_PATH", filepath.Join(root,"audit.jsonl"))
    t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("k",40))
    t.Setenv("AUDIT_SINK_URL","")
    t.Setenv("AUDIT_SINK_TOKEN","")
    var mu sync.Mutex
    var delivered []auditEntry
    rejectSecond := true
    sink := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("s",40) { t.Error("missing sink authentication") }
        var e auditEntry
        if err := json.NewDecoder(r.Body).Decode(&e); err != nil { t.Error(err); w.WriteHeader(400); return }
        mu.Lock()
        defer mu.Unlock()
        if e.Event == "second" && rejectSecond { w.WriteHeader(503); return }
        delivered = append(delivered,e)
        w.WriteHeader(http.StatusNoContent)
    }))
    defer sink.Close()
    configure := func(a *auditLog) {
        a.sinkURL = sink.URL
        a.sinkToken = strings.Repeat("s",40)
        a.sinkClient = sink.Client()
    }

    a, err := openAuditLog()
    if err != nil { t.Fatal(err) }
    configure(a)
    if err := a.Append("GET","/first","owner","first",200); err != nil { t.Fatal(err) }
    savedCursor, err := a.loadReceipt()
    if err != nil || savedCursor != a.previous { t.Fatalf("receipt not durable: %q %v",savedCursor,err) }
    if err := a.Append("POST","/second","owner","second",200); err == nil { t.Fatal("sink failure accepted") }
    if err := a.Append("GET","/third","owner","third",200); err == nil { t.Fatal("appended after failed sink delivery") }
    if err := a.file.Close(); err != nil { t.Fatal(err) }

    // The failed second event was committed locally, but not acknowledged.
    // After restart the HMAC receipt prevents replaying the first event.
    mu.Lock(); rejectSecond = false; mu.Unlock()
    restored, err := openAuditLog()
    if err != nil { t.Fatal(err) }
    configure(restored)
    if err := restored.replayUnacknowledged(); err != nil { t.Fatal(err) }
    if err := restored.Append("GET","/third","owner","third",200); err != nil { t.Fatal(err) }
    mu.Lock()
    if len(delivered) != 3 { t.Fatalf("expected 3 delivered once, got %d",len(delivered)) }
    for i, name := range []string{"first","second","third"} {
        if delivered[i].Event != name { t.Fatalf("event %d = %s, want %s",i,delivered[i].Event,name) }
        if i > 0 && delivered[i].Previous != delivered[i-1].MAC { t.Fatalf("broken audit chain at %d",i) }
    }
    lastMAC := delivered[2].MAC
    mu.Unlock()
    receipt, err := restored.loadReceipt()
    if err != nil || receipt != lastMAC { t.Fatalf("wrong recovery cursor: %s %v",receipt,err) }
    if err := restored.file.Close(); err != nil { t.Fatal(err) }

    // Receipts and full log must remain verifiable on later restarts.
    final, err := openAuditLog()
    if err != nil { t.Fatal(err) }
    _ = final.file.Close()
}

func TestAuditReceiptTamperingAndUnknownMACFailClosed(t *testing.T) {
    root := t.TempDir()
    if err := os.Chmod(root,0700); err != nil { t.Fatal(err) }
    t.Setenv("AUDIT_LOG_PATH",filepath.Join(root,"audit.jsonl"))
    t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("p",40))
    t.Setenv("AUDIT_SINK_URL","")
    a, err := openAuditLog()
    if err != nil { t.Fatal(err) }
    if err := a.Append("GET","/path","owner","first",200); err != nil { t.Fatal(err) }
    if err := a.storeReceipt(a.previous); err != nil { t.Fatal(err) }
    path := a.receiptPath()
    if err := a.file.Close(); err != nil { t.Fatal(err) }
    good, err := os.ReadFile(path)
    if err != nil { t.Fatal(err) }
    if err := os.WriteFile(path,bytes.Replace(good,[]byte("proof"),[]byte("false"),1),0600); err != nil { t.Fatal(err) }
    if bad, err := openAuditLog(); err == nil { _ = bad.file.Close(); t.Fatal("tampered receipt accepted") }
    if err := os.WriteFile(path,good,0600); err != nil { t.Fatal(err) }
    restored, err := openAuditLog()
    if err != nil { t.Fatal(err) }
    if err := restored.storeReceipt(strings.Repeat("a",64)); err != nil { t.Fatal(err) }
    _ = restored.file.Close()
    if bad, err := openAuditLog(); err == nil { _ = bad.file.Close(); t.Fatal("receipt pointing to foreign journal accepted") }
}

func TestAuditStartupFailsClosedUntilPendingSinkDelivery(t *testing.T) {
    root := t.TempDir()
    if err := os.Chmod(root,0700); err != nil { t.Fatal(err) }
    t.Setenv("AUDIT_LOG_PATH",filepath.Join(root,"audit.jsonl"))
    t.Setenv("AUDIT_HMAC_KEY",strings.Repeat("q",40))
    t.Setenv("AUDIT_SINK_URL","")
    a,err:=openAuditLog()
    if err!=nil { t.Fatal(err) }
    if err:=a.Append("POST","/write","owner","started",0);err!=nil { t.Fatal(err) }
    _=a.file.Close()
    t.Setenv("AUDIT_SINK_URL","https://127.0.0.1:1/ingest")
    t.Setenv("AUDIT_SINK_TOKEN",strings.Repeat("s",40))
    if reopened,err:=openAuditLog();err==nil { _=reopened.file.Close(); t.Fatal("startup allowed unreplicated audit event") }
}
