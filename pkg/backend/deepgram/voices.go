package deepgram

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
	"github.com/moeru-ai/unspeech/pkg/utils"
	"github.com/samber/mo"
)

var (
	formats = []types.VoiceFormat{
		{Name: "MP3", Extension: ".mp3", MimeType: "audio/mpeg"},
		{Name: "WAV", Extension: ".wav", MimeType: "audio/wav"},
		{Name: "FLAC", Extension: ".flac", MimeType: "audio/flac"},
		{Name: "AAC", Extension: ".aac", MimeType: "audio/aac"},
		{Name: "OPUS", Extension: ".opus", MimeType: "audio/opus"},
	}
)

type DeepgramModel struct {
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Architecture  string   `json:"architecture"`
	Languages     []string `json:"languages"`
	Version       string   `json:"version"`
	UUID          string   `json:"uuid"`
}

type DeepgramModelsResponse struct {
	TTS []DeepgramModel `json:"tts"`
}

// VoicesCredentials holds the Deepgram API key used to list models.
type VoicesCredentials struct {
	APIKey string
}

// UpstreamError carries a non-2xx response body from Deepgram.
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("deepgram returned %d: %s", e.StatusCode, e.Body)
}

// ListVoices calls Deepgram's /v1/models and maps the TTS catalogue to types.Voice.
func ListVoices(ctx context.Context, creds VoicesCredentials) ([]types.Voice, error) {
	if creds.APIKey == "" {
		return nil, errors.New("deepgram: api key is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepgram.com/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("deepgram: build request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+creds.APIKey)
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deepgram: call models: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(res.Body)
		return nil, &UpstreamError{StatusCode: res.StatusCode, Body: string(body)}
	}

	var response DeepgramModelsResponse

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("deepgram: decode models: %w", err)
	}

	voices := make([]types.Voice, 0, len(response.TTS))

	for _, model := range response.TTS {
		langs := make([]types.VoiceLanguage, len(model.Languages))
		for i, code := range model.Languages {
			langs[i] = types.VoiceLanguage{
				Code:  code,
				Title: code, // Deepgram doesn't provide human-readable titles in the API response
			}
		}

		voices = append(voices, types.Voice{
			ID:               model.CanonicalName,
			Name:             model.Name,
			Languages:        langs,
			Formats:          formats,
			CompatibleModels: []string{model.Architecture}, // e.g., "aura"
			Tags:             make([]string, 0),
			Description:      "Deepgram " + model.Architecture + " voice",
		})
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
			return mo.Err[any](apierrors.
				NewUpstreamError(upErr.StatusCode).
				WithDetail(utils.NewJSONResponseError(upErr.StatusCode, strings.NewReader(upErr.Body)).OrEmpty().Error()))
		}

		return mo.Err[any](apierrors.NewErrBadGateway().WithError(err).WithCaller())
	}

	return mo.Ok[any](types.ListVoicesResponse{Voices: voices})
}
