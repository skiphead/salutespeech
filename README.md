# SaluteSpeech Go Client

[![Go Reference](https://pkg.go.dev/badge/github.com/skiphead/salutespeech.svg)](https://pkg.go.dev/github.com/skiphead/salutespeech)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Production-ready Go client for [SaluteSpeech API](https://developers.sber.ru/docs/ru/salutespeech/overview).

## Features

- 🚀 Full support for sync and async APIs
- 🔐 Automatic token management with caching
- 📝 Configurable logging
- ⏱️ Context support with timeouts
- 🔄 Retry with exponential backoff
- 🎯 Type-safe parameters
- 📦 Clean package structure
- ✅ Comprehensive error handling

## Installation

```bash
go get github.com/skiphead/salutespeech
```

## Quick Start

### Sync Speech Synthesis

```go
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
	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: "base64(client_id:client_secret)",
		Scope:   types.ScopeSaluteSpeechPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	// Create synthesis client
	synthClient, err := sync.NewClient(tokenMgr, sync.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Synthesize speech
	resp, err := synthClient.SynthesizeText(context.Background(),
		"Привет, мир!",
		sync.DefaultOptions())
	if err != nil {
		log.Fatal(err)
	}

	// Save audio
	os.WriteFile("output.wav", resp.AudioData, 0644)
}
```

## Documentation

Full documentation is available at [pkg.go.dev](https://pkg.go.dev/github.com/skiphead/salutespeech).

## Examples

See [examples](./examples) directory for complete examples:

- [Async Recognition](./examples/async_recognition)
- [Sync Recognition](./examples/sync_recognition)
- [Async Synthesis](./examples/async_synthesis)
- [Sync Synthesis](./examples/sync_synthesis)

## License

MIT License - see [LICENSE](LICENSE) file.