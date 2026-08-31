package usage

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"
)

func encodeVarint(n int) []byte {
	var out []byte
	for n > 0x7f {
		out = append(out, byte(n&0x7f)|0x80)
		n >>= 7
	}
	return append(out, byte(n))
}

func encodeLengthDelimited(field int, payload []byte) []byte {
	tag := encodeVarint((field << 3) | 2)
	ln := encodeVarint(len(payload))
	out := make([]byte, 0, len(tag)+len(ln)+len(payload))
	out = append(out, tag...)
	out = append(out, ln...)
	return append(out, payload...)
}

func grpcWebFrame(payload []byte) []byte {
	out := make([]byte, 5+len(payload))
	out[0] = 0
	n := len(payload)
	out[1] = byte(n >> 24)
	out[2] = byte(n >> 16)
	out[3] = byte(n >> 8)
	out[4] = byte(n)
	copy(out[5:], payload)
	return out
}

func encodeRedeemResetRequest(tokenID string) []byte {
	return grpcWebFrame(encodeLengthDelimited(10, []byte(tokenID)))
}

func unwrapGrpcWeb(buf []byte) []byte {
	if len(buf) < 5 {
		return buf
	}
	flag := buf[0]
	ln := int(buf[1])<<24 | int(buf[2])<<16 | int(buf[3])<<8 | int(buf[4])
	if flag&0x7f == 0 && ln >= 0 && 5+ln <= len(buf) {
		return buf[5 : 5+ln]
	}
	return buf
}

func decodeVarint(buf []byte, offset int) (value, next int) {
	shift := 0
	pos := offset
	for pos < len(buf) {
		b := buf[pos]
		pos++
		value |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			return value, pos
		}
		shift += 7
	}
	return value, pos
}

func parseProtobufTimestamp(buf []byte, offset, length int) string {
	end := offset + length
	pos := offset
	seconds := 0
	for pos < end {
		tag, npos := decodeVarint(buf, pos)
		if npos <= pos {
			break
		}
		pos = npos
		field := tag >> 3
		wire := tag & 0x07
		if wire == 0 {
			v, n2 := decodeVarint(buf, pos)
			pos = n2
			if field == 1 {
				seconds = v
			}
		} else {
			break
		}
	}
	if seconds == 0 {
		return ""
	}
	return time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
}

func parseConsumerResetToken(buf []byte) (ResetCredit, bool) {
	id := ""
	exp := ""
	pos := 0
	for pos < len(buf) {
		tag, npos := decodeVarint(buf, pos)
		if npos <= pos {
			break
		}
		pos = npos
		field := tag >> 3
		wire := tag & 0x07
		if wire == 2 {
			ln, n2 := decodeVarint(buf, pos)
			pos = n2
			if pos+ln > len(buf) {
				break
			}
			chunk := buf[pos : pos+ln]
			if field == 10 || field == 1 {
				if s := strings.TrimSpace(string(chunk)); s != "" && utf8.ValidString(s) && len(s) >= 4 && len(s) < 200 {
					id = s
				}
			} else if field == 30 || field == 20 || field == 2 || field == 3 {
				if ts := parseProtobufTimestamp(buf, pos, ln); ts != "" {
					exp = ts
				}
			}
			pos += ln
		} else if wire == 0 {
			_, pos = decodeVarint(buf, pos)
		} else {
			break
		}
	}
	if id == "" {
		return ResetCredit{}, false
	}
	if exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil && !t.After(time.Now()) {
			return ResetCredit{}, false
		}
	}
	return ResetCredit{ID: id, ExpiresAt: exp}, true
}

func walkResetTokens(buf []byte, out *[]ResetCredit) {
	pos := 0
	for pos < len(buf) {
		tag, npos := decodeVarint(buf, pos)
		if npos <= pos {
			break
		}
		pos = npos
		field := tag >> 3
		wire := tag & 0x07
		if wire == 2 {
			ln, n2 := decodeVarint(buf, pos)
			pos = n2
			if pos+ln > len(buf) {
				break
			}
			chunk := buf[pos : pos+ln]
			pos += ln
			if field == 10 || field == 1 {
				if tok, ok := parseConsumerResetToken(chunk); ok {
					*out = append(*out, tok)
				} else {
					walkResetTokens(chunk, out)
				}
			}
		} else if wire == 0 {
			_, pos = decodeVarint(buf, pos)
		} else {
			break
		}
	}
}

func parseRemainingResets(raw []byte) []ResetCredit {
	var out []ResetCredit
	walkResetTokens(unwrapGrpcWeb(raw), &out)
	if len(out) == 0 {
		walkResetTokens(raw, &out)
	}
	return out
}

func parseRemainingResetsJSON(raw []byte) []ResetCredit {
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}
	arr, _ := obj["tokens"].([]any)
	if arr == nil {
		arr, _ = obj["stillRedeemable"].([]any)
	}
	if arr == nil {
		arr, _ = obj["still_redeemable"].([]any)
	}
	var out []ResetCredit
	for _, item := range arr {
		im := mapOf(item)
		if im == nil {
			continue
		}
		id := str(im["tokenId"])
		if id == "" {
			id = str(im["token_id"])
		}
		if id == "" {
			continue
		}
		exp := str(im["validityEnd"])
		if exp == "" {
			exp = str(im["validity_end"])
		}
		if exp != "" {
			if t, err := time.Parse(time.RFC3339, normalizeTime(exp)); err == nil && !t.After(time.Now()) {
				continue
			}
		}
		out = append(out, ResetCredit{ID: id, ExpiresAt: normalizeTime(exp)})
	}
	return out
}
