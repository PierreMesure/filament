package hubspot

import (
	"github.com/galaxy-io/filament"
	"github.com/galaxy-io/filament/registry"
)

func init() {
	registry.RegisterSource("hubspot", filament.MaturityAlpha, func() filament.Source { return New() })
}
