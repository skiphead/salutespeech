package utils

import (
	"encoding/base64"
	"fmt"
)

// GenerateBasicAuthKey encodes client credentials for HTTP Basic Authentication.
// It combines the provided clientID and clientSecret into a "clientID:clientSecret" format
// and returns the result as a base64-encoded string suitable for use in HTTP Basic Auth headers.
//
// Returns an error if either clientID or clientSecret is empty.
// The returned string can be used directly as the value in an "Authorization: Basic <encoded>" header.
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
