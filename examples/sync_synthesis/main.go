package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/synthesis/sync"
	"github.com/skiphead/salutespeech/types"
)

func main() {
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

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	// Create sync synthesis client
	synthClient, err := sync.NewClient(tokenMgr, sync.Config{
		Timeout: 1 * time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Synthesize text
	ctx := context.Background()
	text := "Привет, мир! Это тестовый синтез речи."

	opts := sync.DefaultOptions()
	opts.Voice = types.VoiceMay24000

	resp, err := synthClient.SynthesizeText(ctx, text, opts)
	if err != nil {
		log.Fatal(err)
	}

	// Save audio
	if err := os.WriteFile("output.wav", resp.AudioData, 0644); err != nil {
		log.Fatal(err)
	}

	log.Printf("Audio saved: %d bytes, type: %s", resp.ContentLength, resp.ContentType)
}
