package elevenlabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/moeru-ai/unspeech/pkg/apierrors"
	"github.com/moeru-ai/unspeech/pkg/backend/types"
	"github.com/moeru-ai/unspeech/pkg/utils"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type VoiceLabelKey = string

const (
	VoiceLabelKeyAccent      VoiceLabelKey = "accent"
	VoiceLabelKeyAge         VoiceLabelKey = "age"
	VoiceLabelKeyGender      VoiceLabelKey = "gender"
	VoiceLabelKeyUseCase     VoiceLabelKey = "use_case"
	VoiceLabelKeyDescription VoiceLabelKey = "description"
)

type VoiceSample struct {
	SampleID  string `json:"sample_id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int    `json:"size_bytes"`
	Hash      string `json:"hash"`
}

type VoiceFineTuning struct {
	IsAllowedToFineTune bool `json:"is_allowed_to_fine_tune"`
	State               struct {
		ElevenMultilingualV2 string `json:"eleven_multilingual_v2"`
	} `json:"state"`
	VerificationFailures        []string `json:"verification_failures"`
	VerificationAttemptsCount   int      `json:"verification_attempts_count"`
	ManualVerificationRequested bool     `json:"manual_verification_requested"`
}

type VoiceSettingsKey = string

const (
	VoiceSettingKeyStability       VoiceSettingsKey = "stability"
	VoiceSettingKeySimilarityBoost VoiceSettingsKey = "similarity_boost"
	VoiceSettingKeyStyle           VoiceSettingsKey = "style"
	VoiceSettingKeyUseSpeakerBoost VoiceSettingsKey = "use_speaker_boost"
	VoiceSettingKeySpeed           VoiceSettingsKey = "speed"
)

type VoiceVerifiedLanguage struct {
	Language string `json:"language"`
	ModelID  string `json:"model_id"`
	Accent   string `json:"accent"`
}

type Voice struct {
	VoiceID                 string                   `json:"voice_id"`
	Name                    string                   `json:"name"`
	Category                string                   `json:"category"`
	Labels                  map[VoiceLabelKey]string `json:"labels"`
	AvailableForTiers       []string                 `json:"available_for_tiers"`
	HighQualityBaseModelIds []string                 `json:"high_quality_base_model_ids"`
	Samples                 []VoiceSample            `json:"samples"`
	FineTuning              VoiceFineTuning          `json:"fine_tuning"`
	Description             string                   `json:"description"`
	PreviewURL              string                   `json:"preview_url"`
	Settings                map[VoiceSettingsKey]any `json:"settings"`
	VerifiedLanguages       []VoiceVerifiedLanguage  `json:"verified_languages"`
	SafetyControl           string                   `json:"safety_control"`
}

type ListVoicesResponse struct {
	Voices []Voice `json:"voices"`
}

var (
	// Text to Speech — ElevenLabs Documentation
	// https://elevenlabs.io/docs/capabilities/text-to-speech#supported-languages
	languages = []types.VoiceLanguage{
		{Code: "en-US", Title: "English (USA)"},
		{Code: "en-GB", Title: "English (UK)"},
		{Code: "en-AU", Title: "English (Australia)"},
		{Code: "en-CA", Title: "English (Canada)"},
		{Code: "ja-JP", Title: "Japanese"},
		{Code: "zh-CN", Title: "Chinese"},
		{Code: "de-DE", Title: "German"},
		{Code: "hi-IN", Title: "Hindi"},
		{Code: "fr-FR", Title: "French (France)"},
		{Code: "fr-CA", Title: "French (Canada)"},
		{Code: "ko-KR", Title: "Korean"},
		{Code: "pt-BR", Title: "Portuguese (Brazil)"},
		{Code: "pt-PT", Title: "Portuguese (Portugal)"},
		{Code: "it-IT", Title: "Italian"},
		{Code: "es-ES", Title: "Spanish (Spain)"},
		{Code: "es-MX", Title: "Spanish (Mexico)"},
		{Code: "id-ID", Title: "Indonesian"},
		{Code: "nl-NL", Title: "Dutch"},
		{Code: "tr-TR", Title: "Turkish"},
		{Code: "fil-PH", Title: "Filipino"},
		{Code: "pl-PL", Title: "Polish"},
		{Code: "sv-SE", Title: "Swedish"},
		{Code: "bg-BG", Title: "Bulgarian"},
		{Code: "ro-RO", Title: "Romanian"},
		{Code: "ar-SA", Title: "Arabic (Saudi Arabia)"},
		{Code: "ar-AE", Title: "Arabic (UAE)"},
		{Code: "cs-CZ", Title: "Czech"},
		{Code: "el-GR", Title: "Greek"},
		{Code: "fi-FI", Title: "Finnish"},
		{Code: "hr-HR", Title: "Croatian"},
		{Code: "ms-MY", Title: "Malay"},
		{Code: "sk-SK", Title: "Slovak"},
		{Code: "da-DK", Title: "Danish"},
		{Code: "ta-IN", Title: "Tamil"},
		{Code: "uk-UA", Title: "Ukrainian"},
		{Code: "ru-RU", Title: "Russian"},
	}

	// Flash v2.5 additional languages
	flashV25Languages = []types.VoiceLanguage{
		{Code: "hu-HU", Title: "Hungarian"},
		{Code: "no-NO", Title: "Norwegian"},
		{Code: "vi-VN", Title: "Vietnamese"},
	}

	// Create speech — ElevenLabs Documentation
	// https://elevenlabs.io/docs/api-reference/text-to-speech/convert
	//
	// Text to Speech — ElevenLabs Documentation
	// https://elevenlabs.io/docs/capabilities/text-to-speech#supported-formats
	formats = []types.VoiceFormat{
		// MP3 formats
		{Name: "MP3 22.05kHz 32kbps", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 22050, Bitrate: 32, FormatCode: "mp3_22050_32"},   //nolint:mnd
		{Name: "MP3 44.1kHz 32kbps", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 44100, Bitrate: 32, FormatCode: "mp3_44100_32"},    //nolint:mnd
		{Name: "MP3 44.1kHz 64kbps", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 44100, Bitrate: 64, FormatCode: "mp3_44100_64"},    //nolint:mnd
		{Name: "MP3 44.1kHz 96kbps", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 44100, Bitrate: 96, FormatCode: "mp3_44100_96"},    //nolint:mnd
		{Name: "MP3 44.1kHz 128kbps", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 44100, Bitrate: 128, FormatCode: "mp3_44100_128"}, //nolint:mnd
		{Name: "MP3 44.1kHz 192kbps", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 44100, Bitrate: 192, FormatCode: "mp3_44100_192"}, //nolint:mnd

		// PCM formats (S16LE)
		{Name: "PCM 8kHz", Extension: ".wav", MimeType: "audio/wav", SampleRate: 8000, FormatCode: "pcm_8000"},       //nolint:mnd
		{Name: "PCM 16kHz", Extension: ".wav", MimeType: "audio/wav", SampleRate: 16000, FormatCode: "pcm_16000"},    //nolint:mnd
		{Name: "PCM 22.05kHz", Extension: ".wav", MimeType: "audio/wav", SampleRate: 22050, FormatCode: "pcm_22050"}, //nolint:mnd
		{Name: "PCM 24kHz", Extension: ".wav", MimeType: "audio/wav", SampleRate: 24000, FormatCode: "pcm_24000"},    //nolint:mnd
		{Name: "PCM 44.1kHz", Extension: ".wav", MimeType: "audio/wav", SampleRate: 44100, FormatCode: "pcm_44100"},  //nolint:mnd

		// μ-law format
		{Name: "μ-law 8kHz", Extension: ".ulaw", MimeType: "audio/basic", SampleRate: 8000, FormatCode: "ulaw_8000"}, //nolint:mnd
	}
)

// VoicesCredentials holds the ElevenLabs API key for voices listing.
type VoicesCredentials struct {
	// APIKey is sent upstream as xi-api-key.
	APIKey string
}

// UpstreamError carries a non-2xx response body from the ElevenLabs API.
type UpstreamError struct {
	StatusCode  int
	ContentType string
	Body        string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("elevenlabs returned %d: %s", e.StatusCode, e.Body)
}

// ListVoices fetches the caller's ElevenLabs voices library and maps it to types.Voice.
func ListVoices(ctx context.Context, creds VoicesCredentials) ([]types.Voice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.elevenlabs.io/v1/voices", nil)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: build request: %w", err)
	}

	if creds.APIKey != "" {
		//nolint:canonicalheader
		req.Header.Set("xi-api-key", creds.APIKey)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: call voices: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 400 && res.StatusCode < 600 {
		body, _ := io.ReadAll(res.Body)
		return nil, &UpstreamError{
			StatusCode:  res.StatusCode,
			ContentType: res.Header.Get("Content-Type"),
			Body:        string(body),
		}
	}

	var response ListVoicesResponse

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("elevenlabs: decode voices: %w", err)
	}

	voices := make([]types.Voice, len(response.Voices))

	for i, voice := range response.Voices {
		voices[i] = types.Voice{
			ID:          voice.VoiceID,
			Name:        voice.Name,
			Description: voice.Description,
			Labels: map[string]any{
				types.VoiceLabelKeyAge:    voice.Labels[VoiceLabelKeyAge],
				types.VoiceLabelKeyAccent: voice.Labels[VoiceLabelKeyAccent],
				types.VoiceLabelKeyGender: voice.Labels[VoiceLabelKeyGender],
			},
			Tags:              make([]string, 0),
			Languages:         languages,
			Formats:           formats,
			CompatibleModels:  voice.HighQualityBaseModelIds,
			PredefinedOptions: voice.Settings,
			PreviewAudioURL:   voice.PreviewURL,
		}

		if lo.Contains(voice.HighQualityBaseModelIds, "eleven_flash_v2_5") {
			voices[i].Languages = append(voices[i].Languages, flashV25Languages...)
		}
	}

	return voices, nil
}

func HandleVoices(c echo.Context, _ mo.Option[types.VoicesRequestOptions]) mo.Result[any] {
	creds := VoicesCredentials{
		APIKey: strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer "),
	}

	voices, err := ListVoices(c.Request().Context(), creds)
	if err != nil {
		var upErr *UpstreamError
		if errors.As(err, &upErr) {
			switch {
			case strings.HasPrefix(upErr.ContentType, "application/json"):
				return mo.Err[any](apierrors.
					NewUpstreamError(upErr.StatusCode).
					WithDetail(utils.NewJSONResponseError(upErr.StatusCode, strings.NewReader(upErr.Body)).OrEmpty().Error()))
			case strings.HasPrefix(upErr.ContentType, "text/"):
				return mo.Err[any](apierrors.
					NewUpstreamError(upErr.StatusCode).
					WithDetail(utils.NewTextResponseError(upErr.StatusCode, strings.NewReader(upErr.Body)).OrEmpty().Error()))
			default:
				slog.Warn("unknown upstream error with unknown Content-Type",
					slog.Int("status", upErr.StatusCode),
					slog.String("content_type", upErr.ContentType),
				)

				return mo.Err[any](apierrors.NewUpstreamError(upErr.StatusCode).WithDetail(upErr.Body))
			}
		}

		return mo.Err[any](apierrors.NewErrBadGateway().WithError(err).WithCaller())
	}

	return mo.Ok[any](types.ListVoicesResponse{Voices: voices})
}
