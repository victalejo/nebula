package proxy

import (
	"context"
)

// Slot represents blue or green deployment slot
type Slot string

const (
	SlotBlue  Slot = "blue"
	SlotGreen Slot = "green"
)

// ProxyManager handles reverse proxy configuration
type ProxyManager interface {
	// Route management
	AddRoute(ctx context.Context, route Route) error
	UpdateRoute(ctx context.Context, route Route) error
	RemoveRoute(ctx context.Context, domain string) error
	GetRoute(ctx context.Context, domain string) (*Route, error)
	ListRoutes(ctx context.Context) ([]Route, error)

	// Traffic switching for blue-green deployments
	SwitchTraffic(ctx context.Context, domain string, targetSlot Slot) error

	// SSL management
	ProvisionSSL(ctx context.Context, domain string) error

	// Health
	HealthCheck(ctx context.Context) error
	ReloadConfig(ctx context.Context) error
}

// Route represents a routing configuration
type Route struct {
	Domain      string
	AppID       string
	BlueTarget  *Upstream
	GreenTarget *Upstream
	ActiveSlot  Slot
	SSLEnabled  bool
}

// Upstream represents a backend target
type Upstream struct {
	Host string
	Port int
}

// RouteConfig holds configuration for a route
type RouteConfig struct {
	Domain     string
	TargetHost string
	TargetPort int
	SSLEnabled bool
}
