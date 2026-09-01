package push

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

// recordSize is the single-record size we advertise (RFC 8188); one push
// payload is well under it, so the body is one record.
const recordSize = 4096

// Encrypt produces an `aes128gcm` body (RFC 8291 §3 + RFC 8188) for one
// subscription: p256dh and auth are the browser's keys as base64url from
// PushSubscription.getKey(). A fresh sender key pair and salt per message.
func Encrypt(p256dh, auth string, plaintext []byte) ([]byte, error) {
	uaPub, err := b64.DecodeString(p256dh)
	if err != nil || len(uaPub) != 65 {
		return nil, errors.New("push: bad p256dh key")
	}
	authSecret, err := b64.DecodeString(auth)
	if err != nil || len(authSecret) != 16 {
		return nil, errors.New("push: bad auth secret")
	}
	asPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return encryptWith(uaPub, authSecret, asPriv, salt, plaintext)
}

// encryptWith is Encrypt with the randomness injected — the RFC 8291
// Appendix A vector drives it in the tests.
func encryptWith(uaPub, authSecret []byte, asPriv *ecdh.PrivateKey, salt, plaintext []byte) ([]byte, error) {
	uaKey, err := ecdh.P256().NewPublicKey(uaPub)
	if err != nil {
		return nil, err
	}
	shared, err := asPriv.ECDH(uaKey)
	if err != nil {
		return nil, err
	}
	asPub := asPriv.PublicKey().Bytes()

	// IKM = HKDF(salt=auth, IKM=ecdh, info="WebPush: info\0" || ua_pub || as_pub, 32)
	prkKey, err := hkdf.Extract(sha256.New, shared, authSecret)
	if err != nil {
		return nil, err
	}
	keyInfo := append(append([]byte("WebPush: info\x00"), uaPub...), asPub...)
	ikm, err := hkdf.Expand(sha256.New, prkKey, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	// Content encryption key + nonce (RFC 8188 §2.2 / 2.3), one record.
	prk, err := hkdf.Extract(sha256.New, ikm, salt)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// The last (only) record ends with the 0x02 delimiter; no padding.
	record := append(append([]byte{}, plaintext...), 0x02)
	sealed := gcm.Seal(nil, nonce, record, nil)

	// Header: salt(16) || rs(4) || idlen(1) || keyid(as_pub, 65)
	out := make([]byte, 0, 16+4+1+65+len(sealed))
	out = append(out, salt...)
	out = binary.BigEndian.AppendUint32(out, recordSize)
	out = append(out, byte(len(asPub)))
	out = append(out, asPub...)
	out = append(out, sealed...)
	return out, nil
}

// decrypt is the receiver side, test-only in spirit but kept here so the
// round-trip test exercises exactly the derivation Encrypt used.
func decrypt(uaPriv *ecdh.PrivateKey, authSecret, body []byte) ([]byte, error) {
	if len(body) < 16+4+1+65+17 {
		return nil, errors.New("push: body too short")
	}
	salt := body[:16]
	idlen := int(body[20])
	asPub := body[21 : 21+idlen]
	sealed := body[21+idlen:]
	asKey, err := ecdh.P256().NewPublicKey(asPub)
	if err != nil {
		return nil, err
	}
	shared, err := uaPriv.ECDH(asKey)
	if err != nil {
		return nil, err
	}
	uaPub := uaPriv.PublicKey().Bytes()
	prkKey, _ := hkdf.Extract(sha256.New, shared, authSecret)
	keyInfo := append(append([]byte("WebPush: info\x00"), uaPub...), asPub...)
	ikm, _ := hkdf.Expand(sha256.New, prkKey, string(keyInfo), 32)
	prk, _ := hkdf.Extract(sha256.New, ikm, salt)
	cek, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	record, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, err
	}
	// strip the delimiter and any zero padding after it
	i := len(record) - 1
	for i >= 0 && record[i] == 0 {
		i--
	}
	if i < 0 || (record[i] != 0x02 && record[i] != 0x01) {
		return nil, errors.New("push: bad padding delimiter")
	}
	return record[:i], nil
}
