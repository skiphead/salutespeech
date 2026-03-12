package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/skiphead/salutespeech/pkg" // ⚠️ Замените на реальный путь к модулю
)

// ConsoleLogger реализует интерфейс solutespeech.Logger для вывода в консоль
type ConsoleLogger struct{}

func (l ConsoleLogger) Debug(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, keysAndValues)
}
func (l ConsoleLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, keysAndValues)
}
func (l ConsoleLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, keysAndValues)
}
func (l ConsoleLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, keysAndValues)
}

func main() {
	ctx := context.Background()
	logger := ConsoleLogger{}

	// 1. Получение учетных данных из переменных окружения
	authKey := os.Getenv("SBER_AUTH_KEY")
	if authKey == "" {
		log.Fatal("SBER_AUTH_KEY environment variable is required")
	}

	// 2. Инициализация OAuth клиента
	oauthClient, err := solutespeech.NewClient(solutespeech.Config{
		AuthKey:       authKey,
		Scope:         solutespeech.ScopeSaluteSpeechPers,
		Timeout:       30 * time.Second,
		AllowInsecure: false,
		Logger:        logger,
	})
	if err != nil {
		log.Fatalf("Failed to create OAuth client: %v", err)
	}

	// 3. Инициализация менеджера токенов
	tokenMgr := solutespeech.NewTokenManager(oauthClient, solutespeech.TokenManagerConfig{
		RefreshMargin:      1 * time.Minute,
		MinRefreshInterval: 30 * time.Second,
		Logger:             logger,
	})

	// ==========================================
	// СЦЕНАРИЙ 1: Синхронный синтез речи (Text -> Audio)
	// ==========================================
	fmt.Println("\n=== Scenario 1: Sync Synthesis ===")

	syncSynthClient, err := solutespeech.NewSyncSynthesisClient(tokenMgr, solutespeech.SyncSynthesisClientConfig{
		Timeout:       60 * time.Second,
		AllowInsecure: false,
		Logger:        logger,
	})
	if err != nil {
		log.Fatalf("Failed to create sync synthesis client: %v", err)
	}

	audioResp, err := syncSynthClient.SynthesizeText(ctx, "Привет, это тестовый синтез речи!", solutespeech.DefaultSyncSynthesisOptions())
	if err != nil {
		log.Printf("Synthesis error: %v", err)
	} else {
		fmt.Printf("Synthesis OK: received %d bytes of audio\n", audioResp.ContentLength)
		// Сохраняем аудио в файл
		if err := solutespeech.WriteToFile("output.wav", audioResp.AudioData); err != nil {
			log.Printf("Failed to save audio: %v", err)
		} else {
			fmt.Println("Audio saved to output.wav")
		}
	}

	// ==========================================
	// СЦЕНАРИЙ 2: Асинхронное распознавание (Upload -> Recognize -> Poll)
	// ==========================================
	fmt.Println("\n=== Scenario 2: Async Recognition ===")

	// 2.1. Загрузка файла
	uploadClient, err := solutespeech.NewUploadClient(tokenMgr, "", false, logger)
	if err != nil {
		log.Fatalf("Failed to create upload client: %v", err)
	}

	// ⚠️ Замените на путь к реальному аудиофайлу (>400 байт)
	audioFilePath := "test_audio.wav"
	audioData, err := os.ReadFile(audioFilePath)
	if err != nil {
		log.Printf("Skipping upload (file not found): %v", err)
	} else {
		// ✅ ИСПРАВЛЕНО: Используем существующую константу
		uploadResp, err := uploadClient.Upload(ctx, &solutespeech.UploadRequest{
			Data:        audioData,
			ContentType: solutespeech.ContentAudioPCM16k16bit, // ✅ PCM 16kHz 16-bit
			RequestID:   "",                                   // Генерируется автоматически
		})
		if err != nil {
			log.Printf("Upload error: %v", err)
		} else {
			fmt.Printf("Upload OK: FileID = %s\n", uploadResp.Result.RequestFileID)

			// 2.2. Создание задачи распознавания
			recogClient, err := solutespeech.NewRecognitionClient(tokenMgr, solutespeech.RecognitionClientConfig{
				Timeout:       60 * time.Second,
				AllowInsecure: false,
				Logger:        logger,
			})
			if err != nil {
				log.Fatalf("Failed to create recognition client: %v", err)
			}

			taskResp, err := recogClient.CreateRecognitionTask(ctx, &solutespeech.RecognitionRequest{
				RequestFileID: uploadResp.Result.RequestFileID,
				Options: &solutespeech.RecognitionOptions{
					AudioEncoding: solutespeech.EncodingPCM_S16LE,
					SampleRate:    16000,
					Language:      "ru-RU",
					Model:         solutespeech.ModelGeneral,
				},
			})
			if err != nil {
				log.Printf("Create task error: %v", err)
			} else {
				fmt.Printf("Task created: ID = %s\n", taskResp.Result.ID)

				// 2.3. Ожидание результата
				result, err := recogClient.WaitForResult(ctx, taskResp.Result.ID, 2*time.Second, 5*time.Minute)
				if err != nil {
					log.Printf("Wait result error: %v", err)
				} else {
					fmt.Printf("Recognition done: Status = %s\n", result.UnifiedStatus)
					if len(result.Alternatives) > 0 {
						fmt.Printf("Text: %s\n", result.Alternatives[0].Text)
					}
				}
			}
		}
	}

	// ==========================================
	// СЦЕНАРИЙ 3: Синхронное распознавание (Audio -> Text)
	// ==========================================
	fmt.Println("\n=== Scenario 3: Sync Recognition ===")

	syncRecogClient, err := solutespeech.NewSyncRecognitionClient(tokenMgr, solutespeech.SyncRecognitionClientConfig{
		Timeout:       120 * time.Second,
		AllowInsecure: false,
		Logger:        logger,
	})
	if err != nil {
		log.Fatalf("Failed to create sync recognition client: %v", err)
	}

	// Для синхронного API есть лимиты (файл < 2MB, длительность < 60 сек)
	if len(audioData) > 0 && len(audioData) < 2*1024*1024 {
		// ✅ ИСПРАВЛЕНО: Используем существующую константу
		syncResp, err := syncRecogClient.Recognize(ctx, &solutespeech.SyncRecognitionRequest{
			Data:        audioData,
			ContentType: solutespeech.ContentAudioPCM16k16bit, // ✅ PCM 16kHz 16-bit
			Language:    solutespeech.LangRuRU,
			Model:       solutespeech.ModelMedia,
			SampleRate:  16000,
		})
		if err != nil {
			log.Printf("Sync recognition error: %v", err)
		} else {
			fmt.Printf("Sync Recognition OK: Result = %v\n", syncResp.Result)
		}
	} else {
		fmt.Println("Skipping sync recognition (file too large or missing)")
	}

	fmt.Println("\n=== All scenarios completed ===")
}
