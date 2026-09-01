// Package browserhost is the Chrome native-messaging host for the PiCode
// extension (ADR-0043). Chrome launches this process with stdin/stdout
// length-prefixed JSON; we proxy a small set of calls to the local PiCode
// server discovered via server.json.
package browserhost

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// HostName is the native-messaging host id in Chrome's manifest.
const HostName = "com.picode.browser"

// ExtensionID is pinned by the public key in ext/manifest.json so sideload
// keeps a stable origin for allowed_origins.
const ExtensionID = "beoccbnjejkjjjcmcfhnnklbjaaddolp"

// ExtensionOrigin is what Chrome passes as argv[1] when it launches the host.
const ExtensionOrigin = "chrome-extension://" + ExtensionID + "/"

// WindowsHostExe is the console sibling Chrome actually launches. The tray
// binary is -H=windowsgui and cannot speak native messaging.
const WindowsHostExe = "picode-nmh.exe"

// MaxMessage is Chrome's native-messaging cap (1 MiB). We refuse anything
// larger rather than letting Chrome drop the connection.
const MaxMessage = 1024 * 1024

// IsHostArg reports the two ways Chrome / tests start this process.
func IsHostArg(arg string) bool {
	return arg == "browser-host" || strings.HasPrefix(arg, "chrome-extension://")
}

// Request is one framed message from the extension.
type Request struct {
	Type     string `json:"type"`
	ID       string `json:"id,omitempty"`
	AgentID  string `json:"agentId,omitempty"`
	DeviceID string `json:"deviceId,omitempty"`
	Message  string `json:"message,omitempty"`
	Tab      *Tab   `json:"tab,omitempty"`
	Image    *Image `json:"image,omitempty"`
}

// Tab is the current page the human is looking at.
type Tab struct {
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Selection string `json:"selection,omitempty"`
}

// Image is an opt-in JPEG/PNG of the visible tab (base64, no data: prefix).
type Image struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// Reply is one framed message back to the extension.
type Reply struct {
	OK      bool        `json:"ok"`
	Type    string      `json:"type,omitempty"`
	ID      string      `json:"id,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
	URL     string      `json:"url,omitempty"`
	Agents  []AgentInfo `json:"agents,omitempty"`
	Started bool        `json:"started,omitempty"`
}

// AgentInfo is the compact roster the side panel renders.
type AgentInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Workspace string `json:"workspace,omitempty"`
	Mode      string `json:"mode"`
}

// Handler turns one request into a reply. Tests inject a fake.
type Handler func(Request) Reply

// Serve reads framed requests until EOF and writes a reply for each.
func Serve(in io.Reader, out io.Writer, h Handler) error {
	for {
		req, err := Read(in)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		reply := h(req)
		if reply.ID == "" {
			reply.ID = req.ID
		}
		if reply.Type == "" {
			reply.Type = req.Type
		}
		if err := Write(out, reply); err != nil {
			return err
		}
	}
}

// Read one length-prefixed JSON message (little-endian uint32 + payload).
func Read(r io.Reader) (Request, error) {
	raw, err := readFrame(r)
	if err != nil {
		return Request{}, err
	}
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return Request{}, fmt.Errorf("browserhost: bad request: %w", err)
	}
	return req, nil
}

// Write one length-prefixed JSON message.
func Write(w io.Writer, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(payload) > MaxMessage {
		return fmt.Errorf("browserhost: reply %d bytes exceeds Chrome's 1 MB cap", len(payload))
	}
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

func readFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n == 0 {
		return nil, fmt.Errorf("browserhost: empty message")
	}
	if n > MaxMessage {
		return nil, fmt.Errorf("browserhost: message %d bytes exceeds Chrome's 1 MB cap", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// EncodeFrame is a test helper: one framed payload.
func EncodeFrame(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := Write(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
