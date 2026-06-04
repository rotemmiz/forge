package push

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Sender delivers a Notification to one device's FCM token. Implementations:
//   - fcmSender: the live FCM HTTP v1 client (used when a service account is set).
//
// The interface keeps the dispatcher decoupled from FCM so a future APNs
// implementation (plan 13 open-question #1) slots in without changing the
// dispatcher (plan 13 §"APNs": "design the interface now").
type Sender interface {
	// Send delivers one notification. errUnregistered signals the token is dead
	// and should be pruned; any other error is transient (logged, not pruned).
	Send(ctx context.Context, fcmToken string, n Notification) error
}

// errUnregistered means FCM rejected the token as permanently invalid
// (UNREGISTERED / INVALID_ARGUMENT on the token field). The dispatcher prunes
// the registration so dead tokens stop consuming send attempts.
var errUnregistered = errors.New("push: fcm token unregistered")

// serviceAccount is the subset of a Google service-account JSON key the FCM v1
// OAuth2 exchange needs.
type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// fcmSender is the live FCM HTTP v1 client. It mints a short-lived Google OAuth2
// access token from the service-account key (RS256-signed JWT assertion) and
// posts to projects/{project_id}/messages:send (plan 13 §"Dispatcher").
type fcmSender struct {
	sa         serviceAccount
	signKey    *rsa.PrivateKey
	httpClient *http.Client
	sendURL    string

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// NewFCMSender parses a service-account JSON key and returns a live Sender. A
// malformed or non-service_account key is a hard error so misconfiguration is
// caught at startup rather than silently disabling push.
func NewFCMSender(serviceAccountJSON []byte) (Sender, error) {
	var sa serviceAccount
	if err := json.Unmarshal(serviceAccountJSON, &sa); err != nil {
		return nil, fmt.Errorf("push: parse service account: %w", err)
	}
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("push: credential type %q is not a service_account", sa.Type)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("push: service account missing project_id, client_email, or private_key")
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}
	key, err := parseRSAPrivateKey(sa.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("push: parse private key: %w", err)
	}
	return &fcmSender{
		sa:         sa,
		signKey:    key,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		sendURL:    fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", sa.ProjectID),
	}, nil
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	// Service-account keys are PKCS#8 ("PRIVATE KEY"); accept PKCS#1 too.
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// Send posts one FCM v1 message. The body shape matches plan 13 §"Dispatcher":
// { message: { token, notification{title,body}, data, android{priority:high} } }.
func (f *fcmSender) Send(ctx context.Context, fcmToken string, n Notification) error {
	token, err := f.accessToken(ctx)
	if err != nil {
		return fmt.Errorf("push: oauth2 token: %w", err)
	}
	payload := map[string]any{
		"message": map[string]any{
			"token": fcmToken,
			"notification": map[string]any{
				"title": n.Title,
				"body":  n.Body,
			},
			"data":    n.data(),
			"android": map[string]any{"priority": "high"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("push: marshal fcm message: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.sendURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("push: fcm send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if isUnregistered(resp.StatusCode, respBody) {
		return errUnregistered
	}
	return fmt.Errorf("push: fcm send returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}

// isUnregistered reports whether an FCM error response means the token is
// permanently dead. FCM v1 returns 404 (NOT_FOUND) with errorCode UNREGISTERED,
// or 400 (INVALID_ARGUMENT) for a malformed token.
func isUnregistered(status int, body []byte) bool {
	if status != http.StatusNotFound && status != http.StatusBadRequest {
		return false
	}
	var parsed struct {
		Error struct {
			Status  string `json:"status"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	for _, d := range parsed.Error.Details {
		if d.ErrorCode == "UNREGISTERED" || d.ErrorCode == "INVALID_ARGUMENT" {
			return true
		}
	}
	return parsed.Error.Status == "NOT_FOUND"
}

// accessToken returns a cached Google OAuth2 access token, refreshing it via the
// JWT-bearer grant when missing or within 60s of expiry.
func (f *fcmSender) accessToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.token != "" && time.Until(f.tokenExp) > 60*time.Second {
		return f.token, nil
	}
	assertion, err := f.signJWT()
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.sa.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", errors.New("empty access_token in token response")
	}
	f.token = tr.AccessToken
	f.tokenExp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return f.token, nil
}

// signJWT builds and RS256-signs the service-account JWT assertion for the
// jwt-bearer grant (scope=firebase.messaging, aud=token_uri, 1h expiry).
func (f *fcmSender) signJWT() (string, error) {
	now := time.Now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   f.sa.ClientEmail,
		"scope": fcmScope,
		"aud":   f.sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := b64(headerJSON) + "." + b64(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.signKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
