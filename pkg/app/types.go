package app

import (
	"github.com/BrobridgeOrg/gravity-exporter-stan/pkg/eventbus"
)

type App interface {
	GetEventBus() eventbus.EventBus
}
