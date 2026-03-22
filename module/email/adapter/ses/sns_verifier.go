package ses

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mannaiah/module/email/port"
)

const (
	// defaultSNSMessageVerifierTimeout defines fallback timeout values for sns certificate/confirmation HTTP calls.
	defaultSNSMessageVerifierTimeout = 5 * time.Second
)

// SNSMessageVerifierConfig defines sns-message verifier configuration values.
type SNSMessageVerifierConfig struct {
	// RequestTimeout defines outbound HTTP timeout values.
	RequestTimeout time.Duration
}

// SNSMessageVerifier verifies SNS envelope signatures for SES webhook requests.
type SNSMessageVerifier struct {
	// client defines outbound HTTP dependencies used to fetch signing certificates.
	client *http.Client
}

// NewSNSMessageVerifier creates sns-message verifier dependencies.
func NewSNSMessageVerifier(config SNSMessageVerifierConfig) *SNSMessageVerifier {
	timeout := config.RequestTimeout
	if timeout <= 0 {
		timeout = defaultSNSMessageVerifierTimeout
	}

	return &SNSMessageVerifier{
		client: &http.Client{Timeout: timeout},
	}
}

// Verify validates SNS envelope signature values.
func (v *SNSMessageVerifier) Verify(ctx context.Context, message port.SNSMessage) error {
	if v == nil {
		return errors.New("sns verifier is not configured")
	}
	signingCertURL := strings.TrimSpace(message.SigningCertURL)
	signature := strings.TrimSpace(message.Signature)
	signatureVersion := strings.TrimSpace(message.SignatureVersion)
	if signingCertURL == "" || signature == "" || signatureVersion == "" {
		return errors.New("sns signature metadata is incomplete")
	}

	stringToSign, signErr := buildSNSStringToSign(message)
	if signErr != nil {
		return signErr
	}

	certificate, certErr := v.fetchSigningCertificate(ctx, signingCertURL)
	if certErr != nil {
		return certErr
	}
	publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("sns signing certificate key type is invalid")
	}

	signatureBytes, decodeErr := base64.StdEncoding.DecodeString(signature)
	if decodeErr != nil {
		return fmt.Errorf("decode sns signature: %w", decodeErr)
	}

	hashType, digest, digestErr := digestSNSStringToSign(signatureVersion, stringToSign)
	if digestErr != nil {
		return digestErr
	}
	if verifyErr := rsa.VerifyPKCS1v15(publicKey, hashType, digest, signatureBytes); verifyErr != nil {
		return fmt.Errorf("verify sns signature: %w", verifyErr)
	}

	return nil
}

// fetchSigningCertificate loads one SNS signing certificate from AWS URLs.
func (v *SNSMessageVerifier) fetchSigningCertificate(ctx context.Context, certificateURL string) (*x509.Certificate, error) {
	parsedURL, parseErr := url.Parse(strings.TrimSpace(certificateURL))
	if parseErr != nil {
		return nil, fmt.Errorf("parse sns signing cert url: %w", parseErr)
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return nil, errors.New("sns signing cert url must use https")
	}
	host := strings.ToLower(strings.TrimSpace(parsedURL.Hostname()))
	if host == "" || !strings.HasSuffix(host, ".amazonaws.com") || !strings.Contains(host, "sns") {
		return nil, errors.New("sns signing cert url host is invalid")
	}

	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if requestErr != nil {
		return nil, fmt.Errorf("build sns signing cert request: %w", requestErr)
	}
	response, responseErr := v.client.Do(request)
	if responseErr != nil {
		return nil, fmt.Errorf("request sns signing cert: %w", responseErr)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("request sns signing cert status %d", response.StatusCode)
	}
	pemBytes, readErr := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if readErr != nil {
		return nil, fmt.Errorf("read sns signing cert: %w", readErr)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("decode sns signing cert pem")
	}
	certificate, certErr := x509.ParseCertificate(block.Bytes)
	if certErr != nil {
		return nil, fmt.Errorf("parse sns signing cert: %w", certErr)
	}

	return certificate, nil
}

// digestSNSStringToSign digests one string-to-sign payload according to SNS signature version.
func digestSNSStringToSign(signatureVersion string, stringToSign string) (crypto.Hash, []byte, error) {
	switch strings.TrimSpace(signatureVersion) {
	case "1":
		sum := sha1.Sum([]byte(stringToSign))
		return crypto.SHA1, sum[:], nil
	case "2":
		sum := sha256.Sum256([]byte(stringToSign))
		return crypto.SHA256, sum[:], nil
	default:
		return 0, nil, errors.New("sns signature version is unsupported")
	}
}

// buildSNSStringToSign builds SNS canonical string-to-sign payload values.
func buildSNSStringToSign(message port.SNSMessage) (string, error) {
	messageType := strings.TrimSpace(message.Type)
	if messageType == "" {
		return "", errors.New("sns message type is required")
	}

	parts := make([]string, 0, 14)
	appendPart := func(key string, value string) {
		parts = append(parts, key)
		parts = append(parts, value)
	}

	appendPart("Message", strings.TrimSpace(message.Message))
	appendPart("MessageId", strings.TrimSpace(message.MessageID))
	switch strings.ToLower(messageType) {
	case "notification":
		if strings.TrimSpace(message.Subject) != "" {
			appendPart("Subject", strings.TrimSpace(message.Subject))
		}
	case "subscriptionconfirmation", "unsubscribeconfirmation":
		appendPart("SubscribeURL", strings.TrimSpace(message.SubscribeURL))
		appendPart("Timestamp", strings.TrimSpace(message.Timestamp))
		appendPart("Token", strings.TrimSpace(message.Token))
	default:
		return "", errors.New("sns message type is unsupported")
	}
	if !strings.EqualFold(messageType, "subscriptionconfirmation") &&
		!strings.EqualFold(messageType, "unsubscribeconfirmation") {
		appendPart("Timestamp", strings.TrimSpace(message.Timestamp))
	}
	appendPart("TopicArn", strings.TrimSpace(message.TopicARN))
	appendPart("Type", messageType)

	for index := 0; index < len(parts); index += 2 {
		if strings.TrimSpace(parts[index+1]) == "" {
			return "", fmt.Errorf("sns canonical field %s is required", parts[index])
		}
	}

	var builder strings.Builder
	for index := 0; index < len(parts); index += 2 {
		builder.WriteString(parts[index])
		builder.WriteByte('\n')
		builder.WriteString(parts[index+1])
		builder.WriteByte('\n')
	}

	return builder.String(), nil
}
