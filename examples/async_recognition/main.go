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

	// Upload file
	uploadClient, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		log.Fatal(err)
	}

	uploadResp, err := uploadClient.UploadFromFile(context.Background(), "audio.wav",
		types.ContentAudioPCM16k16bit)
	if err != nil {
		log.Fatal(err)
	}

	// Create recognition task
	recClient, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		log.Fatal(err)
	}

	req := &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID,
		Options: &async.Options{
			AudioEncoding: async.EncodingPCM_S16LE,
			SampleRate:    16000,
			Model:         async.ModelGeneral,
			Language:      "ru-RU",
		},
	}

	resp, err := recClient.CreateTask(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task created: %s\n", resp.Result.ID)

	// Wait for result
	result, err := recClient.WaitForResult(context.Background(), resp.Result.ID, 2*time.Second, 5*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Recognition result: %+v\n", result.Alternatives)
}
