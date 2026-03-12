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

func main() {
	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: "base64(client_id:client_secret)", // Replace with your credentials
		Scope:   types.ScopeSaluteSpeechPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	// Upload text file
	uploadClient, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		log.Fatal(err)
	}

	text := "Привет, мир! Это тестовый синтез речи."
	textFile := "text.txt"
	if err := os.WriteFile(textFile, []byte(text), 0644); err != nil {
		log.Fatal(err)
	}
	defer os.Remove(textFile)

	uploadResp, err := uploadClient.UploadFromFile(context.Background(), textFile,
		types.ContentTextPlain)
	if err != nil {
		log.Fatal(err)
	}

	// Create synthesis task
	synthClient, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		log.Fatal(err)
	}

	req := &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID,
		AudioEncoding: async.EncodingOpus,
		Voice:         types.VoiceMay24000,
	}

	resp, err := synthClient.CreateTask(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task created: %s\n", resp.Result.ID)

	// Wait for result and download audio
	result, err := synthClient.WaitForTask(context.Background(), resp.Result.ID,
		2*time.Second, 5*time.Minute, true)
	if err != nil {
		log.Fatal(err)
	}

	// Save audio
	if err := os.WriteFile("output.opus", result.AudioData, 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Audio saved: %d bytes\n", len(result.AudioData))
}
