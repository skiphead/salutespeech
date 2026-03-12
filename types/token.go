package types

import "time"

// Token represents OAuth token
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// IsValid checks if token is valid with given margin
func (t *Token) IsValid(margin time.Duration) bool {
	if t == nil || t.Value == "" {
		return false
	}
	return time.Now().Add(margin).Before(t.ExpiresAt)
}

// TokenResponse represents OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
