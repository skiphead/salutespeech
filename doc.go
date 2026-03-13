/*
Package salutespeech provides a comprehensive Go client for SaluteSpeech API by Sber.

Basic usage:

	// Create OAuth client
	authKey := salutespeech.GenerateBasicAuthKey("client_id", "client_secret")
	client, err := salutespeech.NewClient(salutespeech.Config{
	    AuthKey: authKey, // or Authorization Key
	    Scope:   salutespeech.ScopeSaluteSpeechPers,
	})

	// Create token manager
	tokenMgr := salutespeech.NewTokenManager(client, salutespeech.TokenManagerConfig{})

	// Use sync synthesis
	synthClient, _ := salutespeech.NewSyncSynthesisClient(tokenMgr, salutespeech.SyncSynthesisClientConfig{})
	resp, _ := synthClient.SynthesizeText(ctx, "Hello world", salutespeech.DefaultSyncSynthesisOptions())

Features:
  - Automatic token management with caching
  - Sync and async API support
  - Configurable logging
  - Context support with timeouts
  - Retry with exponential backoff
  - Type-safe parameters
*/
package salutespeech
