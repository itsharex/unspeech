package alibaba

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/moeru-ai/unspeech/pkg/apierrors"
	"github.com/moeru-ai/unspeech/pkg/backend/types"
	"github.com/samber/mo"
)

var (
	//go:embed voices.json
	voicesJSON string
)

type VoicesResponseItem struct {
	Name            string   `json:"name"`
	PreviewAudioURL string   `json:"preview_audio_url"`
	Model           string   `json:"model"`
	Voice           string   `json:"voice"`
	Scenarios       []string `json:"scenarios"`
	Language        string   `json:"language"`
	Bitrate         string   `json:"bitrate"`
	Format          string   `json:"format"`
}

// ListVoices returns Alibaba's static voice catalogue embedded at build time.
// No credentials are required because Alibaba does not expose a public voices/list API.
func ListVoices(_ context.Context) ([]types.Voice, error) {
	var voicesResponse []VoicesResponseItem

	if err := json.Unmarshal([]byte(voicesJSON), &voicesResponse); err != nil {
		return nil, fmt.Errorf("alibaba: decode embedded voices: %w", err)
	}

	voices := make([]types.Voice, 0, len(voicesResponse))

	for _, voice := range voicesResponse {
		voices = append(voices, types.Voice{
			ID:          voice.Voice,
			Name:        voice.Name,
			Description: voice.Name,
			Labels: map[string]any{
				"tailoredScenarios": voice.Scenarios,
			},
			Tags: make([]string, 0),
			Formats: []types.VoiceFormat{
				// https://www.volcengine.com/docs/6561/1257584
				{Name: "MP3", Extension: ".mp3", MimeType: "audio/mp3", SampleRate: 8000, Bitrate: 16, FormatCode: "mp3"},  //nolint:mnd
				{Name: "MP3", Extension: ".mp3", MimeType: "audio/mp3", SampleRate: 16000, Bitrate: 16, FormatCode: "mp3"}, //nolint:mnd
				{Name: "MP3", Extension: ".mp3", MimeType: "audio/mp3", SampleRate: 22050, Bitrate: 16, FormatCode: "mp3"}, //nolint:mnd
				{Name: "MP3", Extension: ".mp3", MimeType: "audio/mp3", SampleRate: 24000, Bitrate: 16, FormatCode: "mp3"}, //nolint:mnd
				{Name: "MP3", Extension: ".mp3", MimeType: "audio/mp3", SampleRate: 44100, Bitrate: 16, FormatCode: "mp3"}, //nolint:mnd
				{Name: "MP3", Extension: ".mp3", MimeType: "audio/mp3", SampleRate: 48000, Bitrate: 16, FormatCode: "mp3"}, //nolint:mnd
				{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 8000, Bitrate: 16, FormatCode: "pcm"},  //nolint:mnd
				{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 16000, Bitrate: 16, FormatCode: "pcm"}, //nolint:mnd
				{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 22050, Bitrate: 16, FormatCode: "pcm"}, //nolint:mnd
				{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 24000, Bitrate: 16, FormatCode: "pcm"}, //nolint:mnd
				{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 44100, Bitrate: 16, FormatCode: "pcm"}, //nolint:mnd
				{Name: "PCM", Extension: ".pcm", MimeType: "audio/pcm", SampleRate: 48000, Bitrate: 16, FormatCode: "pcm"}, //nolint:mnd
				{Name: "WAV", Extension: ".wav", MimeType: "audio/wav", SampleRate: 8000, Bitrate: 16, FormatCode: "wav"},  //nolint:mnd
				{Name: "WAV", Extension: ".wav", MimeType: "audio/wav", SampleRate: 16000, Bitrate: 16, FormatCode: "wav"}, //nolint:mnd
				{Name: "WAV", Extension: ".wav", MimeType: "audio/wav", SampleRate: 22050, Bitrate: 16, FormatCode: "wav"}, //nolint:mnd
				{Name: "WAV", Extension: ".wav", MimeType: "audio/wav", SampleRate: 24000, Bitrate: 16, FormatCode: "wav"}, //nolint:mnd
				{Name: "WAV", Extension: ".wav", MimeType: "audio/wav", SampleRate: 44100, Bitrate: 16, FormatCode: "wav"}, //nolint:mnd
				{Name: "WAV", Extension: ".wav", MimeType: "audio/wav", SampleRate: 48000, Bitrate: 16, FormatCode: "wav"}, //nolint:mnd
			},
			CompatibleModels: []string{voice.Model},
			PreviewAudioURL:  voice.PreviewAudioURL,
			Languages: []types.VoiceLanguage{
				{
					Title: voice.Language,
					Code:  voice.Language,
				},
			},
		})
	}

	return voices, nil
}

func HandleVoices(c echo.Context, _ mo.Option[types.VoicesRequestOptions]) mo.Result[any] {
	voices, err := ListVoices(c.Request().Context())
	if err != nil {
		return mo.Err[any](apierrors.NewErrInternal().WithDetail(err.Error()).WithCaller())
	}

	return mo.Ok[any](types.ListVoicesResponse{Voices: voices})
}
