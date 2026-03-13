package utils

import (
	"encoding/base64"
	"fmt"
)

// GenerateBasicAuthKey encodes client credentials for HTTP Basic Auth.
// Returns base64-encoded string of "clientID:clientSecret".
func GenerateBasicAuthKey(clientID, clientSecret string) (string, error) {
	if clientID == "" {
		return "", fmt.Errorf("clientID cannot be empty")
	}
	if clientSecret == "" {
		return "", fmt.Errorf("clientSecret cannot be empty")
	}

	cred := fmt.Sprintf("%s:%s", clientID, clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(cred)), nil
}
