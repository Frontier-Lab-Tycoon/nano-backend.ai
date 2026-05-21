package service

import (
	"github.com/seedspirit/nano-backend.ai/internal/manager/service/runsvc"
)

// Services groups manager service dependencies for handlers.
type Services struct {
	RunSvc *runsvc.Service
}

// NewServices creates an empty service registry.
func NewServices() *Services {
	return &Services{}
}

// WithRunService registers the run service.
func (s *Services) WithRunService(args runsvc.Args) *Services {
	s.RunSvc = runsvc.NewService(args)
	return s
}
