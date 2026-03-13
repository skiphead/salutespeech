// Package main provides an example of using the SaluteSpeech API client library
// for asynchronous speech recognition. It demonstrates the complete workflow:
// authentication, file upload, task creation, and result retrieval.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/recognition/async"
	"github.com/skiphead/salutespeech/types"
	"github.com/skiphead/salutespeech/upload"
)

func main() {
	// Generate Basic Authentication credentials from client ID and secret
	// These credentials are used to obtain OAuth tokens from the SaluteSpeech API
	authKey := client.GenerateBasicAuthKey("client_id", "client_secret")

	// Create OAuth client for token management
	// The OAuth client handles the authentication flow and token retrieval
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: authKey,                     // Base64-encoded client credentials
		Scope:   types.ScopeSaluteSpeechPers, // API access scope
		Timeout: 30 * time.Second,            // HTTP client timeout
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager for automatic token refresh
	// The token manager handles token caching, refresh, and provides valid tokens for API requests
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	// Upload audio file to SaluteSpeech storage
	// Files must be uploaded before they can be processed by async recognition
	uploadClient, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Upload audio file from local filesystem
	// The file should be in PCM 16kHz 16-bit format (mono) for optimal recognition
	uploadResp, err := uploadClient.UploadFromFile(context.Background(), "audio.wav",
		types.ContentAudioPCM16k16bit)
	if err != nil {
		log.Fatal(err)
	}

	// Create asynchronous recognition task
	// The recognition client handles task creation and status polling
	recClient, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Configure recognition options
	// These parameters control the speech recognition behavior
	req := &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID, // ID of uploaded audio file
		Options: &async.Options{
			AudioEncoding: async.EncodingPCM_S16LE, // Audio encoding format
			SampleRate:    16000,                   // Audio sample rate in Hz
			Model:         async.ModelGeneral,      // Recognition model to use
			Language:      "ru-RU",                 // Language of the audio
		},
	}

	// Submit recognition task to the API
	// Returns task ID for tracking progress
	resp, err := recClient.CreateTask(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task created: %s\n", resp.Result.ID)

	// Wait for recognition task completion
	// Polls task status until completion, error, or timeout
	// Parameters: poll interval (2s) and maximum wait time (5min)
	result, err := recClient.WaitForResult(context.Background(), resp.Result.ID, 2*time.Second, 5*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	// Display recognition results
	// Alternatives contain transcribed text with confidence scores
	fmt.Printf("Recognition result: %+v\n", result.Alternatives)
}
