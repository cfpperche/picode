package push

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("b64 %q: %v", s, err)
	}
	return b
}

// RFC 8291 Appendix A: the one interoperability vector every Web Push
// library checks itself against.
func TestEncryptMatchesRFC8291Vector(t *testing.T) {
	plaintext := []byte("When I grow up, I want to be a watermelon")
	uaPub := mustB64(t, "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4")
	auth := mustB64(t, "BTBZMqHH6r4Tts7J_aSIgg")
	asPrivRaw := mustB64(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw")
	salt := mustB64(t, "DGv6ra1nlYgDCS1FRnbzlw")
	want := mustB64(t, "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN")
	asPriv, err := ecdh.P256().NewPrivateKey(asPrivRaw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := encryptWith(uaPub, auth, asPriv, salt, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if base64.RawURLEncoding.EncodeToString(got) != base64.RawURLEncoding.EncodeToString(want) {
		t.Fatalf("ciphertext mismatch\n got %s\nwant %s", base64.RawURLEncoding.EncodeToString(got), base64.RawURLEncoding.EncodeToString(want))
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	uaPriv, err := ecdh.P256().GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	for i := range auth {
		auth[i] = byte(i * 7)
	}
	msg := []byte(`{"title":"builder needs you","body":"Run the migration?","hash":"#/agent/a1"}`)
	body, err := Encrypt(base64.RawURLEncoding.EncodeToString(uaPriv.PublicKey().Bytes()), base64.RawURLEncoding.EncodeToString(auth), msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 86 || body[20] != 65 {
		t.Fatalf("header shape wrong: len=%d idlen=%d", len(body), body[20])
	}
	back, err := decrypt(uaPriv, auth, body)
	if err != nil {
		t.Fatal(err)
	}
	if string(back) != string(msg) {
		t.Fatalf("round trip = %q", back)
	}
	if _, err := Encrypt("nope", "x", msg); err == nil {
		t.Fatal("bad keys must be refused")
	}
}

func TestVapidKeysPersistAndSign(t *testing.T) {
	dir := t.TempDir()
	k1, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "vapid.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	k2, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if k1.PublicKey() != k2.PublicKey() {
		t.Fatal("reload must give the same key")
	}
	if pub := mustB64(t, k1.PublicKey()); len(pub) != 65 || pub[0] != 0x04 {
		t.Fatalf("public key = %d bytes, first %x", len(pub), pub[0])
	}

	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	hdr, err := k1.Authorization("https://fcm.googleapis.com/fcm/send/abc", "mailto:a@b.c", now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hdr, "vapid t=") || !strings.Contains(hdr, ", k="+k1.PublicKey()) {
		t.Fatalf("header = %s", hdr)
	}
	jwt := strings.TrimPrefix(strings.SplitN(hdr, ",", 2)[0], "vapid t=")
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt parts = %d", len(parts))
	}
	var claims map[string]any
	if err := json.Unmarshal(mustB64(t, parts[1]), &claims); err != nil {
		t.Fatal(err)
	}
	if claims["aud"] != "https://fcm.googleapis.com" || claims["sub"] != "mailto:a@b.c" || int64(claims["exp"].(float64)) != now.Add(12*time.Hour).Unix() {
		t.Fatalf("claims = %v", claims)
	}
	sig := mustB64(t, parts[2])
	if len(sig) != 64 {
		t.Fatalf("sig len = %d", len(sig))
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(&k1.priv.PublicKey, sum[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Fatal("signature does not verify")
	}
	if _, err := k1.Authorization("not a url", "mailto:x", now); err == nil {
		t.Fatal("bad endpoint must be refused")
	}
}

func TestSenderPostsAndMapsGone(t *testing.T) {
	keys, _ := LoadOrCreate(t.TempDir())
	uaPriv, _ := ecdh.P256().GenerateKey(nil)
	auth := make([]byte, 16)
	var seen http.Header
	var gotBody []byte
	status := 201
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		gotBody = buf[:n]
		w.WriteHeader(status)
	}))
	defer srv.Close()
	s := &Sender{Keys: keys, Subject: "mailto:t@example.com"}
	target := Target{Endpoint: srv.URL + "/sub/1", P256dh: base64.RawURLEncoding.EncodeToString(uaPriv.PublicKey().Bytes()), Auth: base64.RawURLEncoding.EncodeToString(auth)}
	if err := s.Send(context.Background(), target, []byte(`{"title":"hi"}`), 300, "high", "ask:a1"); err != nil {
		t.Fatal(err)
	}
	if seen.Get("Content-Encoding") != "aes128gcm" || seen.Get("TTL") != "300" || seen.Get("Urgency") != "high" || seen.Get("Topic") != "ask:a1" || !strings.HasPrefix(seen.Get("Authorization"), "vapid t=") {
		t.Fatalf("headers = %v", seen)
	}
	back, err := decrypt(uaPriv, auth, gotBody)
	if err != nil || string(back) != `{"title":"hi"}` {
		t.Fatalf("decrypt = %q, %v", back, err)
	}
	status = 410
	if err := s.Send(context.Background(), target, []byte("x"), 1, "", ""); err != ErrGone {
		t.Fatalf("410 → %v, want ErrGone", err)
	}
	status = 500
	if err := s.Send(context.Background(), target, []byte("x"), 1, "", ""); err == nil || err == ErrGone {
		t.Fatalf("500 → %v", err)
	}
}
