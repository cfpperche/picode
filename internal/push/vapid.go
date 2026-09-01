// Package push sends Web Push notifications (RFC 8030 / 8291 / 8292) with
// nothing but the standard library: an ECDSA P-256 VAPID key pair, HKDF +
// AES-128-GCM message encryption, and one HTTPS POST per subscription.
// ADR-0046: the phone (ADR-0044) needs to be called when an agent is
// blocked on a decision or a run finishes while nobody is at the machine.
package push

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Keys is the server's VAPID identity. One per install, kept in DataDir;
// rotating it invalidates every subscription (browsers bind to the key).
type Keys struct {
	priv *ecdsa.PrivateKey
}

type keyFile struct {
	D string `json:"d"` // base64url, 32-byte scalar
	X string `json:"x"`
	Y string `json:"y"`
}

var b64 = base64.RawURLEncoding

// LoadOrCreate reads <dir>/vapid.json, or generates a key and writes it
// with mode 0600. The file is the only secret push depends on.
func LoadOrCreate(dir string) (*Keys, error) {
	path := filepath.Join(dir, "vapid.json")
	if raw, err := os.ReadFile(path); err == nil {
		var kf keyFile
		if err := json.Unmarshal(raw, &kf); err == nil {
			if k, err := keysFromFile(kf); err == nil {
				return k, nil
			}
		}
		// Unreadable: regenerate rather than run without push, but keep
		// the broken file beside the new one for inspection.
		_ = os.Rename(path, path+".broken")
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	k := &Keys{priv: priv}
	kf := keyFile{D: b64.EncodeToString(pad32(priv.D)), X: b64.EncodeToString(pad32(priv.X)), Y: b64.EncodeToString(pad32(priv.Y))}
	raw, _ := json.MarshalIndent(kf, "", "  ")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return nil, err
	}
	return k, nil
}

func keysFromFile(kf keyFile) (*Keys, error) {
	d, err := b64.DecodeString(kf.D)
	if err != nil || len(d) != 32 {
		return nil, errors.New("vapid: bad private scalar")
	}
	priv := new(ecdsa.PrivateKey)
	priv.Curve = elliptic.P256()
	priv.D = new(big.Int).SetBytes(d)
	priv.X, priv.Y = priv.Curve.ScalarBaseMult(d)
	if kf.X != "" {
		x, _ := b64.DecodeString(kf.X)
		if new(big.Int).SetBytes(x).Cmp(priv.X) != 0 {
			return nil, errors.New("vapid: public point does not match scalar")
		}
	}
	return &Keys{priv: priv}, nil
}

func pad32(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// PublicKey is the uncompressed P-256 point (65 bytes) — what
// PushManager.subscribe wants as applicationServerKey — base64url.
func (k *Keys) PublicKey() string {
	return b64.EncodeToString(k.publicPoint())
}

func (k *Keys) publicPoint() []byte {
	return append([]byte{0x04}, append(pad32(k.priv.X), pad32(k.priv.Y)...)...)
}

// Authorization builds the RFC 8292 header for one push-service endpoint:
// `vapid t=<ES256 JWT>, k=<public key>`. aud is the endpoint's origin,
// exp 12h (the maximum the spec allows is 24h), sub identifies the sender.
func (k *Keys) Authorization(endpoint, subject string, now time.Time) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("vapid: bad endpoint %q", endpoint)
	}
	header := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, _ := json.Marshal(map[string]any{
		"aud": u.Scheme + "://" + u.Host,
		"exp": now.Add(12 * time.Hour).Unix(),
		"sub": subject,
	})
	signing := header + "." + b64.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, k.priv, sum[:])
	if err != nil {
		return "", err
	}
	// JWS wants the raw R||S (64 bytes), not ASN.1.
	sig := append(pad32(r), pad32(s)...)
	return "vapid t=" + signing + "." + b64.EncodeToString(sig) + ", k=" + k.PublicKey(), nil
}
