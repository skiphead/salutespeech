// Package main provides an example of using the SaluteSpeech API client library
// for synchronous speech recognition. It demonstrates a complete workflow:
// authentication, client setup, and real-time audio file recognition.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	client2 "github.com/skiphead/salutespeech/pkg/client"
	"github.com/skiphead/salutespeech/recognition/sync"
	"github.com/skiphead/salutespeech/types"
)

func main() {
	// Generate Basic Authentication credentials from client ID and secret
	// These credentials are used to obtain OAuth tokens from the SaluteSpeech API
	authKey := client2.GenerateBasicAuthKey("client_id", "client_secret")

	// Create OAuth client for token management
	// The OAuth client handles the authentication flow and token retrieval
	oauthClient, err := client2.NewOAuthClient(client2.Config{
		AuthKey: authKey,                     // Base64-encoded client credentials
		Scope:   types.ScopeSaluteSpeechPers, // API access scope for speech recognition
		Timeout: 30 * time.Second,            // HTTP client timeout
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager for automatic token refresh
	// The token manager handles token caching, refresh, and provides valid tokens for API requests
	tokenMgr := client2.NewTokenManager(oauthClient, client2.TokenManagerConfig{})

	// Create synchronous speech recognition client
	// Sync recognition is suitable for short audio files (max 2MB) and returns results immediately
	recClient, err := sync.NewClient(tokenMgr, sync.Config{
		Timeout: 2 * time.Minute, // Extended timeout for longer audio processing
	})
	if err != nil {
		log.Fatal(err)
	}

	// Recognize audio from file using synchronous recognition
	// The audio file should be in PCM 16kHz 16-bit format (mono) for optimal results
	ctx := context.Background()
	resp, err := recClient.RecognizeFromFile(ctx, "audio.wav",
		types.ContentAudioPCM16k16bit, // Content type for PCM 16kHz 16-bit audio
		sync.DefaultOptions())         // Use default recognition options
	if err != nil {
		log.Fatal(err)
	}

	// Print recognition results
	// Result contains transcribed text with confidence scores
	fmt.Printf("Recognition results: %+v\n", resp.Result)

	// Print emotion analysis if available
	// Emotions field provides sentiment analysis of the recognized speech
	if len(resp.Emotions) > 0 {
		fmt.Printf("Emotions: %+v\n", resp.Emotions)
	}
}
