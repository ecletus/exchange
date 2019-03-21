package exchange

import "github.com/ecletus/plug"

var E_RESOURCE_ADDED = PKG + ".resourceAdded"

type ResourceAddedEvent struct {
	plug.EventInterface
	Exchange *Exchange
	Resource *Resource
}

type dispatcher struct {
	dis plug.EventDispatcherInterface
}

func (dis *dispatcher) OnResourceAdded(cb func(e *ResourceAddedEvent)) *dispatcher {
	dis.dis.On(E_RESOURCE_ADDED, func(e plug.EventInterface) {
		cb(e.(*ResourceAddedEvent))
	})
	return dis
}

func Dis(dis plug.EventDispatcherInterface) *dispatcher {
	return &dispatcher{dis}
}
