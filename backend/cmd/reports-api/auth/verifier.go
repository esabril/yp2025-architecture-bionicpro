package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AuthVerifier struct {
	issuer  string
	jwksURL string
	client  *http.Client
	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
}

func NewVerifier(issuer, jwksURL string) *AuthVerifier {
	return &AuthVerifier{
		issuer:  issuer,
		jwksURL: jwksURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		keys:    map[string]*rsa.PublicKey{},
	}
}

func (v *AuthVerifier) Verify(ctx context.Context, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("token must have three parts")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, err
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Claims{}, err
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return Claims{}, errors.New("unsupported token header")
	}

	publicKey, err := v.key(ctx, header.Kid)
	if err != nil {
		return Claims{}, err
	}

	signed := []byte(parts[0] + "." + parts[1])
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, err
	}
	digest := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return Claims{}, err
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var tokenClaims Claims
	if err := json.Unmarshal(claimsBytes, &tokenClaims); err != nil {
		return Claims{}, err
	}
	if tokenClaims.Exp <= time.Now().Unix() {
		return Claims{}, errors.New("token expired")
	}
	if tokenClaims.Iss != v.issuer {
		return Claims{}, fmt.Errorf("unexpected issuer %q", tokenClaims.Iss)
	}
	if tokenClaims.PreferredUsername == "" {
		return Claims{}, errors.New("preferred_username is missing")
	}
	return tokenClaims, nil
}

func (v *AuthVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key := v.keys[kid]
	v.mu.RUnlock()
	if key != nil {
		return key, nil
	}

	if err := v.refresh(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key = v.keys[kid]
	if key == nil {
		return nil, fmt.Errorf("key %q not found", kid)
	}
	return key, nil
}

func (v *AuthVerifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("jwks status %d", resp.StatusCode)
	}

	var source jwks
	if err := json.NewDecoder(resp.Body).Decode(&source); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(source.Keys))
	for _, item := range source.Keys {
		if item.Kty != "RSA" || item.N == "" || item.E == "" {
			continue
		}
		key, err := v.rsaPublicKey(item.N, item.E)
		if err != nil {
			return err
		}
		keys[item.Kid] = key
	}

	v.mu.Lock()
	v.keys = keys
	v.mu.Unlock()
	return nil
}

func (v *AuthVerifier) rsaPublicKey(nValue, eValue string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nValue)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eValue)
	if err != nil {
		return nil, err
	}

	exponent := 0
	for _, b := range eBytes {
		exponent = exponent*256 + int(b)
	}
	if exponent == 0 {
		return nil, errors.New("invalid rsa exponent")
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}
