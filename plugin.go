package exchange

import (
	"github.com/ecletus/plug"
	"github.com/moisespsena-go/aorm"
)

type Plugin struct {
	ExchangeKey string
	FakeDBKey   string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.FakeDBKey}
}

func (p *Plugin) ProvideOptions() []string {
	return []string{p.ExchangeKey}
}

func (p *Plugin) Init(options *plug.Options) {
	options.Set(p.ExchangeKey, &Exchange{
		FakeDB:    options.GetInterface(p.FakeDBKey).(*aorm.DB),
		resources: map[string]*Resource{},
	})
}
