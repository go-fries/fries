package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// HeaderID contains the unique webhook message ID.
	HeaderID = "Webhook-Id"
	// HeaderTimestamp contains the message creation time as Unix seconds.
	HeaderTimestamp = "Webhook-Timestamp"
	// HeaderSignature contains one or more versioned message signatures.
	HeaderSignature = "Webhook-Signature"

	signatureVersion = "v1"
)

// Signer creates Standard Webhooks HMAC-SHA256 signatures.
//
// A Signer is safe for concurrent use.
type Signer struct {
	secrets [][]byte
	now     func() time.Time
}

// NewSigner creates a Signer with secret as its primary signing secret.
func NewSigner(secret Secret, options ...SignerOption) (*Signer, error) {
	c, err := newSignerConfig(secret, options...)
	if err != nil {
		return nil, err
	}

	return &Signer{
		secrets: c.secrets,
		now:     time.Now,
	}, nil
}

// Sign signs messageID, the current Unix timestamp, and the exact payload
// bytes. It returns complete Webhook headers ready to add to an HTTP request.
//
// Sign returns [ErrInvalidMessageID] unless messageID contains only visible
// ASCII characters other than a full stop.
func (s *Signer) Sign(
	messageID string,
	payload []byte,
) (http.Header, error) {
	if !validMessageID(messageID) {
		return nil, ErrInvalidMessageID
	}

	timestamp := strconv.FormatInt(s.now().Unix(), 10)
	signatures := make([]string, 0, len(s.secrets))
	for _, secret := range s.secrets {
		digest := signDigest(secret, messageID, timestamp, payload)
		signatures = append(
			signatures,
			signatureVersion+","+base64.StdEncoding.EncodeToString(digest),
		)
	}

	headers := make(http.Header, 3)
	headers.Set(HeaderID, messageID)
	headers.Set(HeaderTimestamp, timestamp)
	headers.Set(HeaderSignature, strings.Join(signatures, " "))
	return headers, nil
}

func signDigest(
	secret []byte,
	messageID string,
	timestamp string,
	payload []byte,
) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = io.WriteString(mac, messageID)
	_, _ = mac.Write([]byte{'.'})
	_, _ = io.WriteString(mac, timestamp)
	_, _ = mac.Write([]byte{'.'})
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func validMessageID(messageID string) bool {
	if messageID == "" {
		return false
	}
	for i := range len(messageID) {
		value := messageID[i]
		if value < '!' || value > '~' || value == '.' {
			return false
		}
	}
	return true
}
