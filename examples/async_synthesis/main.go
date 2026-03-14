// Package main provides an example of using the SaluteSpeech API client library
// for asynchronous speech synthesis (text-to-speech). It demonstrates the complete workflow:
// authentication, text file upload, task creation, result polling, and audio download.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/synthesis/async"
	"github.com/skiphead/salutespeech/types"
	"github.com/skiphead/salutespeech/upload"
)

const savePath = "output.opus"

func main() {
	// Generate Basic Authentication credentials from client ID and secret
	// These credentials are used to obtain OAuth tokens from the SaluteSpeech API
	authKey := client.GenerateBasicAuthKey("client_id", "client_secret")

	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: authKey,
		Scope:   types.ScopeSaluteSpeechPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager for automatic token refresh
	// The token manager handles token caching, refresh, and provides valid tokens for API requests
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	// Upload text file to SaluteSpeech storage
	// For async synthesis, text must be uploaded as a file before processing
	uploadClient, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Prepare text for synthesis min size 400 bytes
	text := `Через сто лет человечество, вероятно, достигнет технологической сингулярности. 
             Искусственный интеллект превзойдет человека, но станет ли он нашим партнером или конкурентом? 
             Медицина победит старение, превратив 120 лет в новый средний возраст. 
             Люди колонизируют Луну и Марс, а энергия термоядерного синтеза решит проблему ресурсов.
             Однако главный вопрос останется этическим: сохраним ли мы человечность в мире киборгов и цифрового бессмертия? 
             Сотрем ли границы между реальностью и виртуальностью? 
             Возможно, наши потомки будут с ностальгией вспоминать XXI век как время простых радостей и неразгаданных тайн.`
	textFile := "text.txt"

	// Create temporary text file
	if err := os.WriteFile(textFile, []byte(text), 0644); err != nil {
		log.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Fatal(err)
		}
	}(textFile) // Clean up temporary file

	// Upload text file to the API
	// Returns a file ID that will be used in the synthesis request
	uploadResp, err := uploadClient.UploadFromFile(context.Background(), textFile,
		types.ContentTextPlain) // Content type for plain text
	if err != nil {
		log.Fatal(err)
	}

	// Create asynchronous synthesis client
	// This client handles task creation, status checking, and result retrieval
	synthClient, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Configure synthesis request
	req := &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID, // ID of uploaded text file
		AudioEncoding: async.EncodingOpus,              // Output audio format (Opus)
		Voice:         types.VoiceMay24000,             // Voice to use for synthesis
	}

	// Submit synthesis task to the API
	// Returns task ID for tracking progress
	resp, err := synthClient.CreateTask(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task created: %s\n", resp.Result.ID)

	// Wait for synthesis task completion and download audio
	// Parameters:
	//   - poll interval: 2 seconds between status checks
	//   - maximum wait time: 5 minutes
	//   - downloadAudio: true (automatically download when complete)
	result, err := synthClient.WaitForTask(context.Background(), resp.Result.ID,
		2*time.Second, 5*time.Minute, true)
	if err != nil {
		log.Fatal(err)
	}

	// Save synthesized audio to file
	// The audio format matches the requested EncodingOpus
	if err := os.WriteFile(savePath, result.AudioData, 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Audio saved: %d bytes\n", len(result.AudioData))
}
