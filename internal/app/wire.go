//go:build wireinject

package app

import (
	"log/slog"
	"net/http"

	"github.com/google/wire"
	"github.com/gowvp/gb28181/internal/conf"
	"github.com/gowvp/gb28181/internal/data"
	"github.com/gowvp/gb28181/internal/web/api"
)

func wireApp(bc *conf.Bootstrap, log *slog.Logger) (http.Handler, func(), error) {
	panic(wire.Build(data.ProviderSet, api.ProviderVersionSet, api.ProviderSet))
}
