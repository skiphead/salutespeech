package types

// Logger interface for logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// NoopLogger implements Logger with no operations
type NoopLogger struct{}

func (n NoopLogger) Debug(string, ...interface{}) {}
func (n NoopLogger) Info(string, ...interface{})  {}
func (n NoopLogger) Warn(string, ...interface{})  {}
func (n NoopLogger) Error(string, ...interface{}) {}

// Scope defines API access scope
type Scope string

const (
	ScopeSaluteSpeechPers Scope = "SALUTE_SPEECH_PERS"
	ScopeSaluteSpeechCorp Scope = "SALUTE_SPEECH_CORP"
	ScopeSaluteSpeechB2B  Scope = "SALUTE_SPEECH_B2B"
	ScopeSberSpeech       Scope = "SBER_SPEECH"
	ScopeGigaChatAPIPers  Scope = "GIGACHAT_API_PERS"
)

// Voice represents synthesis voice
type Voice string

const (
	VoiceOst24000     Voice = "Ost_24000"
	VoiceAida24000    Voice = "Aida_24000"
	VoiceFilipp24000  Voice = "Filipp_24000"
	VoiceJasmine24000 Voice = "Jasmine_24000"
	VoiceMay24000     Voice = "May_24000"
	VoiceErmil24000   Voice = "Ermil_24000"
	VoiceMay8000      Voice = "May_8000"
	VoiceOst8000      Voice = "Ost_8000"
	VoiceJoy24000     Voice = "Joy_24000"
	VoiceNick24000    Voice = "Nick_24000"
	VoiceAigerim24000 Voice = "Aigerim_24000"
	VoiceNazira24000  Voice = "Nazira_24000"
)
