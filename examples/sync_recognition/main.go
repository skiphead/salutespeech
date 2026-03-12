package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/recognition/sync"
	"github.com/skiphead/salutespeech/types"
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

	// Create sync recognition client
	recClient, err := sync.NewClient(tokenMgr, sync.Config{
		Timeout: 2 * time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Recognize from file
	ctx := context.Background()
	resp, err := recClient.RecognizeFromFile(ctx, "audio.wav",
		types.ContentAudioPCM16k16bit,
		sync.DefaultOptions())
	if err != nil {
		log.Fatal(err)
	}

	// Print results
	fmt.Printf("Recognition results: %+v\n", resp.Result)
	if len(resp.Emotions) > 0 {
		fmt.Printf("Emotions: %+v\n", resp.Emotions)
	}
}
