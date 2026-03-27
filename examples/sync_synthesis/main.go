// Package main provides an example of using the SaluteSpeech API client library
// for synchronous speech synthesis (text-to-speech). It demonstrates a complete workflow:
// authentication, client setup, and real-time text-to-speech conversion.
package main

import (
	"context"
	"log"
	"os"
	"time"

	client2 "github.com/skiphead/salutespeech/pkg/client"
	"github.com/skiphead/salutespeech/synthesis/sync"
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
		Scope:   types.ScopeSaluteSpeechPers, // API access scope for speech synthesis
		Timeout: 30 * time.Second,            // HTTP client timeout
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager for automatic token refresh
	// The token manager handles token caching, refresh, and provides valid tokens for API requests
	tokenMgr := client2.NewTokenManager(oauthClient, client2.TokenManagerConfig{})

	// Create synchronous speech synthesis client
	// Sync synthesis converts text to speech in real-time and returns audio immediately
	// Suitable for short texts and real-time applications
	synthClient, err := sync.NewClient(tokenMgr, sync.Config{
		Timeout: 1 * time.Minute, // Timeout for synthesis request
	})
	if err != nil {
		log.Fatal(err)
	}

	// Prepare text for synthesis
	ctx := context.Background()
	text := "Привет, мир! Это тестовый синтез речи." // "Hello, world! This is a test speech synthesis."

	// Configure synthesis options
	opts := sync.DefaultOptions()    // Start with default options (WAV format, default voice)
	opts.Voice = types.VoiceMay24000 // Use specific voice (May, 24kHz)

	// Perform synchronous text-to-speech conversion
	// The text is sent to the API and audio is returned immediately
	resp, err := synthClient.SynthesizeText(ctx, text, opts)
	if err != nil {
		log.Fatal(err)
	}

	// Save synthesized audio to file
	// Default format is WAV (16kHz, 16-bit) unless specified otherwise in options
	if err := os.WriteFile("output.wav", resp.AudioData, 0644); err != nil {
		log.Fatal(err)
	}

	// Log information about the synthesized audio
	log.Printf("Audio saved: %d bytes, type: %s", resp.ContentLength, resp.ContentType)
}
