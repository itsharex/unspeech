package koemotion

import (
	"context"
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/moeru-ai/unspeech/pkg/apierrors"
	"github.com/moeru-ai/unspeech/pkg/backend/types"
	"github.com/samber/mo"
)

// ErrNotImplemented is returned by ListVoices until Koemotion provides a catalogue.
var ErrNotImplemented = errors.New("koemotion: voices listing not implemented")

// ListVoices is a stub placeholder — Koemotion has no voices API yet.
func ListVoices(_ context.Context) ([]types.Voice, error) {
	return nil, ErrNotImplemented
}

func HandleVoices(_ echo.Context, _ mo.Option[types.VoicesRequestOptions]) mo.Result[any] {
	return mo.Err[any](apierrors.NewErrNotFound().WithDetail("not implemented"))
}
