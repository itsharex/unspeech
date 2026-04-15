package microsoft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/moeru-ai/unspeech/pkg/apierrors"
	"github.com/moeru-ai/unspeech/pkg/backend/types"
	"github.com/nekomeowww/xo"
	"github.com/samber/mo"
)

type VoiceTagKey string

const (
	VoiceTagKeyTailoredScenarios  VoiceTagKey = "TailoredScenarios"
	VoiceTagKeyVoicePersonalities VoiceTagKey = "VoicePersonalities"
)

var (
	// https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech
	formats = []types.VoiceFormat{
		// AMR-WB
		{Name: "AMR-WB", Extension: ".amr", MimeType: "audio/amr-wb", SampleRate: 16000, FormatCode: "amr-wb-16000hz"}, //nolint:mnd

		// Opus formats
		{Name: "Opus", Extension: ".opus", MimeType: "audio/opus", SampleRate: 16000, Bitrate: 32, FormatCode: "audio-16khz-16bit-32kbps-mono-opus"}, //nolint:mnd
		{Name: "Opus", Extension: ".opus", MimeType: "audio/opus", SampleRate: 24000, Bitrate: 24, FormatCode: "audio-24khz-16bit-24kbps-mono-opus"}, //nolint:mnd
		{Name: "Opus", Extension: ".opus", MimeType: "audio/opus", SampleRate: 24000, Bitrate: 48, FormatCode: "audio-24khz-16bit-48kbps-mono-opus"}, //nolint:mnd

		// MP3 formats
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 16000, Bitrate: 32, FormatCode: "audio-16khz-32kbitrate-mono-mp3"},   //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 16000, Bitrate: 64, FormatCode: "audio-16khz-64kbitrate-mono-mp3"},   //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 16000, Bitrate: 128, FormatCode: "audio-16khz-128kbitrate-mono-mp3"}, //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 24000, Bitrate: 48, FormatCode: "audio-24khz-48kbitrate-mono-mp3"},   //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 24000, Bitrate: 96, FormatCode: "audio-24khz-96kbitrate-mono-mp3"},   //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 24000, Bitrate: 160, FormatCode: "audio-24khz-160kbitrate-mono-mp3"}, //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 48000, Bitrate: 96, FormatCode: "audio-48khz-96kbitrate-mono-mp3"},   //nolint:mnd
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg", SampleRate: 48000, Bitrate: 192, FormatCode: "audio-48khz-192kbitrate-mono-mp3"}, //nolint:mnd

		// G722
		{Name: "G722", Extension: ".g722", MimeType: "audio/g722", SampleRate: 16000, Bitrate: 64, FormatCode: "g722-16khz-64kbps"}, //nolint:mnd

		// Ogg Opus formats
		{Name: "Ogg Opus", Extension: ".ogg", MimeType: "audio/ogg", SampleRate: 16000, FormatCode: "ogg-16khz-16bit-mono-opus"}, //nolint:mnd
		{Name: "Ogg Opus", Extension: ".ogg", MimeType: "audio/ogg", SampleRate: 24000, FormatCode: "ogg-24khz-16bit-mono-opus"}, //nolint:mnd
		{Name: "Ogg Opus", Extension: ".ogg", MimeType: "audio/ogg", SampleRate: 48000, FormatCode: "ogg-48khz-16bit-mono-opus"}, //nolint:mnd

		// Raw PCM formats
		{Name: "PCM A-law", Extension: ".alaw", MimeType: "audio/x-alaw-basic", SampleRate: 8000, FormatCode: "raw-8khz-8bit-mono-alaw"},   //nolint:mnd
		{Name: "PCM μ-law", Extension: ".ulaw", MimeType: "audio/x-mulaw-basic", SampleRate: 8000, FormatCode: "raw-8khz-8bit-mono-mulaw"}, //nolint:mnd
		{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 8000, FormatCode: "raw-8khz-16bit-mono-pcm"},                   //nolint:mnd
		{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 16000, FormatCode: "raw-16khz-16bit-mono-pcm"},                 //nolint:mnd
		{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 22050, FormatCode: "raw-22050hz-16bit-mono-pcm"},               //nolint:mnd
		{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 24000, FormatCode: "raw-24khz-16bit-mono-pcm"},                 //nolint:mnd
		{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 44100, FormatCode: "raw-44100hz-16bit-mono-pcm"},               //nolint:mnd
		{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 48000, FormatCode: "raw-48khz-16bit-mono-pcm"},                 //nolint:mnd

		// TrueSilk formats
		{Name: "TrueSilk", Extension: ".silk", MimeType: "audio/silk", SampleRate: 16000, FormatCode: "raw-16khz-16bit-mono-truesilk"}, //nolint:mnd
		{Name: "TrueSilk", Extension: ".silk", MimeType: "audio/silk", SampleRate: 24000, FormatCode: "raw-24khz-16bit-mono-truesilk"}, //nolint:mnd

		// WebM formats
		{Name: "WebM Opus", Extension: ".webm", MimeType: "audio/webm", SampleRate: 16000, FormatCode: "webm-16khz-16bit-mono-opus"},                     //nolint:mnd
		{Name: "WebM Opus", Extension: ".webm", MimeType: "audio/webm", SampleRate: 24000, Bitrate: 24, FormatCode: "webm-24khz-16bit-24kbps-mono-opus"}, //nolint:mnd
		{Name: "WebM Opus", Extension: ".webm", MimeType: "audio/webm", SampleRate: 24000, FormatCode: "webm-24khz-16bit-mono-opus"},                     //nolint:mnd
	}
)

type Voice struct {
	Name            string                `json:"Name"`
	DisplayName     string                `json:"DisplayName"`
	LocalName       string                `json:"LocalName"`
	ShortName       string                `json:"ShortName"`
	Gender          string                `json:"Gender"`
	Locale          string                `json:"Locale"`
	LocaleName      string                `json:"LocaleName"`
	SampleRateHertz string                `json:"SampleRateHertz"`
	VoiceType       string                `json:"VoiceType"`
	Status          string                `json:"Status"`
	VoiceTag        map[VoiceTagKey][]any `json:"VoiceTag"`
	WordsPerMinute  string                `json:"WordsPerMinute"`
}

// masterpiecePreviewURL builds the Azure "Masterpieces" sample audio URL for a voice.
// The blob key is derived from ShortName by stripping the trailing "Neural" suffix
// (traditional voices like en-US-JennyNeural -> en-US-Jenny) and removing the ":"
// separator that newer HD/Dragon voice ShortNames embed
// (zh-CN-Xiaoxiao2:DragonHDFlashLatestNeural -> zh-CN-Xiaoxiao2DragonHDFlashLatest).
// Using DisplayName here would URL-encode spaces and 404 against the blob service.
func masterpiecePreviewURL(shortName string) string {
	slug := strings.TrimSuffix(shortName, "Neural")
	slug = strings.ReplaceAll(slug, ":", "")

	return "https://ai.azure.com/speechassetscache/ttsvoice/Masterpieces/" + slug + "-General-Audio.wav"
}

// VoicesCredentials are the per-request inputs needed to hit Azure Speech voices/list.
type VoicesCredentials struct {
	// Region is an Azure region identifier, e.g. "eastasia".
	Region string
	// SubscriptionKey is the Ocp-Apim-Subscription-Key value.
	SubscriptionKey string
}

// UpstreamError represents a non-2xx response from Azure Speech.
// Callers can errors.As() this to surface upstream status to their own HTTP clients.
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("microsoft speech returned %d: %s", e.StatusCode, e.Body)
}

// ListVoices calls Azure Speech "voices/list" and maps the response to unspeech's types.Voice.
func ListVoices(ctx context.Context, creds VoicesCredentials) ([]types.Voice, error) {
	if creds.Region == "" {
		return nil, errors.New("microsoft: region is required")
	}

	if creds.SubscriptionKey == "" {
		return nil, errors.New("microsoft: subscription key is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://"+creds.Region+".tts.speech.microsoft.com/cognitiveservices/voices/list", nil)
	if err != nil {
		return nil, fmt.Errorf("microsoft: build request: %w", err)
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", creds.SubscriptionKey)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoft: call voices/list: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 400 && res.StatusCode < 600 {
		body, _ := io.ReadAll(res.Body)
		return nil, &UpstreamError{StatusCode: res.StatusCode, Body: string(body)}
	}

	var response []Voice

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("microsoft: decode voices: %w", err)
	}

	voices := make([]types.Voice, 0, len(response))

	for _, voice := range response {
		tags := make([]string, 0, len(voice.VoiceTag[VoiceTagKeyTailoredScenarios])+len(voice.VoiceTag[VoiceTagKeyVoicePersonalities]))

		for _, tag := range voice.VoiceTag[VoiceTagKeyTailoredScenarios] {
			tags = append(tags, xo.Stringify(tag))
		}

		for _, tag := range voice.VoiceTag[VoiceTagKeyVoicePersonalities] {
			tags = append(tags, xo.Stringify(tag))
		}

		voices = append(voices, types.Voice{
			ID:          voice.ShortName,
			Name:        voice.DisplayName,
			Description: voice.Name,
			Labels: map[string]any{
				types.VoiceLabelKeyType:   voice.VoiceType,
				types.VoiceLabelKeyAccent: voice.LocalName,
				types.VoiceLabelKeyGender: voice.Gender,
				"tailoredScenarios":       voice.VoiceTag[VoiceTagKeyTailoredScenarios],
				"voicePersonalities":      voice.VoiceTag[VoiceTagKeyVoicePersonalities],
			},
			Tags:             tags,
			Formats:          formats,
			CompatibleModels: []string{"v1"},
			PreviewAudioURL: masterpiecePreviewURL(voice.ShortName),
			Languages: []types.VoiceLanguage{
				{
					Title: voice.LocalName,
					Code:  voice.Locale,
				},
			},
		})
	}

	return voices, nil
}

func HandleVoices(c echo.Context, options mo.Option[types.VoicesRequestOptions]) mo.Result[any] {
	region := options.MustGet().ExtraQuery.Get("region")
	if region == "" {
		region = "eastasia"
	}

	creds := VoicesCredentials{
		Region:          region,
		SubscriptionKey: strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer "),
	}

	voices, err := ListVoices(c.Request().Context(), creds)
	if err != nil {
		var upErr *UpstreamError
		if errors.As(err, &upErr) {
			res := &http.Response{
				StatusCode: upErr.StatusCode,
				Body:       io.NopCloser(strings.NewReader(upErr.Body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}

			if resError := handleResponseError(res); resError.IsError() {
				return resError
			}
		}

		return mo.Err[any](apierrors.NewErrInternal().WithError(err).WithCaller())
	}

	return mo.Ok[any](types.ListVoicesResponse{Voices: voices})
}
