package kalshi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const kalshiWSPath = "/trade-api/ws/v2"

func buildAuthHeaders(apiKeyID string, privateKeyPath string) (http.Header, error) {
	privateKeyPEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read kalshi private key file: %w", err)
	}

	privateKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	timestampMs := strconv.FormatInt(time.Now().UnixMilli(), 10)
	message := timestampMs + "GET" + kalshiWSPath

	signature, err := signRSAPSS(privateKey, []byte(message))
	if err != nil {
		return nil, err
	}

	header := http.Header{}
	header.Set("KALSHI-ACCESS-KEY", apiKeyID)
	header.Set("KALSHI-ACCESS-TIMESTAMP", timestampMs)
	header.Set("KALSHI-ACCESS-SIGNATURE", signature)

	return header, nil
}

func signRSAPSS(privateKey *rsa.PrivateKey, message []byte) (string, error) {
	digest := sha256.Sum256(message)

	sig, err := rsa.SignPSS(
		rand.Reader,
		privateKey,
		crypto.SHA256,
		digest[:],
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("rsa-pss sign kalshi websocket request: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

func parseRSAPrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("decode kalshi private key PEM: no PEM block found")
	}

	parsedPKCS8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := parsedPKCS8.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("kalshi private key is PKCS8 but not RSA")
		}
		return rsaKey, nil
	}

	parsedPKCS1, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return parsedPKCS1, nil
	}

	return nil, fmt.Errorf("parse kalshi private key: unsupported PEM type %q", strings.TrimSpace(block.Type))
}
