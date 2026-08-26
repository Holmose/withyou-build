package managedcontext

// Module aggregates managed context HTTP handlers.
type Module struct {
	Handler *Handler
}

func NewModule(handler *Handler) *Module {
	return &Module{Handler: handler}
}
