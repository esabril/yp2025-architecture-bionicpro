package auth

type Claims struct {
	Exp               int64  `json:"exp"`
	Iss               string `json:"iss"`
	PreferredUsername string `json:"preferred_username"`
	Subject           string `json:"sub"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}
