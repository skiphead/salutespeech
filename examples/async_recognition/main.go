// Package main provides an example of using the SaluteSpeech API client library
// for asynchronous speech recognition. It demonstrates the complete workflow:
// authentication, file upload with automatic format detection, task creation,
// result retrieval, and downloading the transcribed text.
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
	"github.com/skiphead/salutespeech/utils"
)

func main() {
	// Generate Basic Authentication credentials from client ID and secret
	// These credentials are used to obtain OAuth tokens from the SaluteSpeech API
	authKey := client.GenerateBasicAuthKey("client_id", "client_secret")

	// Create OAuth client for token management
	// The OAuth client handles the authentication flow and token retrieval
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: authKey,                     // Base64-encoded client credentials
		Scope:   types.ScopeSaluteSpeechPers, // API access scope for speech recognition
		Timeout: 30 * time.Second,            // HTTP client timeout
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager for automatic token refresh
	// The token manager handles token caching, refresh, and provides valid tokens for API requests
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	// Initialize upload client for file uploads
	// Files must be uploaded to SaluteSpeech storage before async processing
	uploadClient, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Specify path to audio file for recognition
	pathAudioFile := "tests/audio.ogg"

	// Automatically detect audio format by reading file header and extension
	// This utility function determines the correct content type for the API
	audioType, detectErr := utils.DetectAudioContentType(pathAudioFile)
	if detectErr != nil {
		log.Fatal(detectErr)
	}

	// Upload audio file to SaluteSpeech storage
	// The detected content type ensures proper format specification
	uploadResp, err := uploadClient.UploadFromFile(context.Background(), pathAudioFile,
		audioType)
	if err != nil {
		log.Fatal(err)
	}

	// Create asynchronous recognition client
	// This client handles task creation, status polling, and result retrieval
	recClient, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Configure recognition options for the audio file
	// Note: AudioEncoding must match the actual format of the uploaded file (OGG_OPUS)
	req := &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID, // ID from upload response
		Options: &async.Options{
			AudioEncoding: async.EncodingOGG_OPUS, // Must match the actual audio format
			SampleRate:    16000,                  // Audio sample rate in Hz
			Model:         async.ModelGeneral,     // General-purpose recognition model
			Language:      "ru-RU",                // Russian language
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
	// Polls task status every 2 seconds with a 5-minute timeout
	result, err := recClient.WaitForResult(context.Background(), resp.Result.ID, 2*time.Second, 5*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Recognition result file ID: %+v\n", result.Result.ResponseFileId)

	// Create another client instance for downloading the result
	// (In practice, you could reuse the existing client)
	downloadTextFile, _ := async.NewClient(tokenMgr, async.Config{})

	// Download the transcribed text to a local file
	// The result contains the recognized text in JSON format
	err = downloadTextFile.DownloadTaskResultToFile(context.Background(), result.Result.ResponseFileId, "./test.txt")
	if err != nil {
		fmt.Println(err)
	}
}
