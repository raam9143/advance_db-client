package server

import (
    "bufio"
    "bytes"
    "encoding/hex"
    "encoding/json"
    "errors"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// The cursor is an authenticated high-water mark. It is advanced only after
// the receiver acknowledges an event. On a crash between delivery and cursor
// sync, replay may deliver the event again: receivers MUST deduplicate by MAC.
type auditReceipt struct {
    MAC string `json:"mac"`
    Proof string `json:"proof"`
}

func (a *auditLog) receiptPath() string { return a.file.Name() + ".receipt" }

func (a *auditLog) loadReceipt() (string, error) {
    path := a.receiptPath()
    info, err := os.Lstat(path)
    if os.IsNotExist(err) { return "", nil }
    if err != nil { return "", err }
    if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 1024 {
        return "", errors.New("audit receipt must be a small owner-private regular file")
    }
    data, err := os.ReadFile(path)
    if err != nil { return "", err }
    reader := json.NewDecoder(bytes.NewReader(data))
    reader.DisallowUnknownFields()
    var r auditReceipt
    if err := reader.Decode(&r); err != nil { return "", errors.New("invalid audit receipt") }
    var extra any
    if err := reader.Decode(&extra); err != io.EOF { return "", errors.New("invalid trailing audit receipt data") }
    raw, err := hex.DecodeString(r.MAC)
    if err != nil || len(raw) != 32 || strings.ToLower(r.MAC) != r.MAC || r.Proof != digest(a.key, "audit-receipt-v1:"+r.MAC) {
        return "", errors.New("audit receipt integrity verification failed")
    }
    return r.MAC, nil
}

func (a *auditLog) storeReceipt(mac string) error {
    path := a.receiptPath()
    parent := filepath.Dir(path)
    tmp, err := os.CreateTemp(parent, ".audit-receipt-*")
    if err != nil { return err }
    tmpPath := tmp.Name()
    defer os.Remove(tmpPath)
    receipt := auditReceipt{MAC:mac, Proof:digest(a.key,"audit-receipt-v1:"+mac)}
    if err := json.NewEncoder(tmp).Encode(receipt); err != nil { _ = tmp.Close(); return err }
    if err := tmp.Sync(); err != nil { _ = tmp.Close(); return err }
    if err := tmp.Close(); err != nil { return err }
    if err := os.Rename(tmpPath, path); err != nil { return err }
    dir, err := os.Open(parent)
    if err != nil { return err }
    defer dir.Close()
    return dir.Sync()
}

// replayUnacknowledged validates the receipt against the already-verified
// journal and re-sends only events after that cursor. Production receivers
// must durably acknowledge and idempotently deduplicate on event MAC.
func (a *auditLog) replayUnacknowledged() error {
    acknowledged, err := a.loadReceipt()
    if err != nil { return err }
    if _, err := a.file.Seek(0, io.SeekStart); err != nil { return err }
    scanner := bufio.NewScanner(a.file)
    scanner.Buffer(make([]byte,16*1024), 1<<20)
    found := acknowledged == ""
    for scanner.Scan() {
        var e auditEntry
        if err := json.Unmarshal(scanner.Bytes(), &e); err != nil { return err }
        if !found {
            if e.MAC == acknowledged { found = true }
            continue
        }
        if a.sinkURL == "" { continue }
        if err := a.deliver(scanner.Bytes()); err != nil { return err }
        if err := a.storeReceipt(e.MAC); err != nil { return err }
    }
    if err := scanner.Err(); err != nil { return err }
    if !found { return errors.New("audit receipt not found in verified journal") }
    _, err = a.file.Seek(0, io.SeekEnd)
    return err
}
