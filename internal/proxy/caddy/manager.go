package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/proxy"
)

// Manager implements the ProxyManager interface for Caddy
type Manager struct {
	adminAPI string
	network  string
	client   *http.Client
	log      logger.Logger
}

// NewManager creates a new Caddy manager
func NewManager(adminAPI string, network string, log logger.Logger) *Manager {
	return &Manager{
		adminAPI: adminAPI,
		network:  network,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log,
	}
}

// CaddyConfig represents the Caddy JSON configuration
type CaddyConfig struct {
	Apps CaddyApps `json:"apps"`
}

type CaddyApps struct {
	HTTP CaddyHTTP `json:"http"`
}

type CaddyHTTP struct {
	Servers map[string]*CaddyServer `json:"servers"`
}

type CaddyServer struct {
	Listen []string      `json:"listen"`
	Routes []CaddyRoute  `json:"routes"`
}

type CaddyRoute struct {
	Match   []CaddyMatch   `json:"match,omitempty"`
	Handle  []CaddyHandler `json:"handle"`
	Terminal bool          `json:"terminal,omitempty"`
}

type CaddyMatch struct {
	Host []string `json:"host,omitempty"`
}

type CaddyHandler struct {
	Handler   string           `json:"handler"`
	Upstreams []CaddyUpstream  `json:"upstreams,omitempty"`
	Routes    []CaddyRoute     `json:"routes,omitempty"`
}

type CaddyUpstream struct {
	Dial string `json:"dial"`
}

// AddRoute adds a new route to Caddy
func (m *Manager) AddRoute(ctx context.Context, route proxy.Route) error {
	m.log.Info("adding route to caddy",
		"domain", route.Domain,
		"app_id", route.AppID,
	)

	// Get current active upstream
	var upstream *proxy.Upstream
	if route.ActiveSlot == proxy.SlotBlue {
		upstream = route.BlueTarget
	} else {
		upstream = route.GreenTarget
	}

	if upstream == nil {
		return fmt.Errorf("no active upstream configured")
	}

	// Create reverse proxy route
	caddyRoute := CaddyRoute{
		Match: []CaddyMatch{
			{Host: []string{route.Domain}},
		},
		Handle: []CaddyHandler{
			{
				Handler: "reverse_proxy",
				Upstreams: []CaddyUpstream{
					{Dial: fmt.Sprintf("%s:%d", upstream.Host, upstream.Port)},
				},
			},
		},
		Terminal: true,
	}

	// Add route via Caddy Admin API
	routeJSON, err := json.Marshal(caddyRoute)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", m.adminAPI)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(routeJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy returned error %d: %s", resp.StatusCode, string(body))
	}

	m.log.Info("route added successfully",
		"domain", route.Domain,
	)

	return nil
}

// UpdateRoute updates an existing route
func (m *Manager) UpdateRoute(ctx context.Context, route proxy.Route) error {
	// For simplicity, remove and re-add the route
	if err := m.RemoveRoute(ctx, route.Domain); err != nil {
		m.log.Warn("failed to remove old route", "error", err)
	}
	return m.AddRoute(ctx, route)
}

// RemoveRoute removes a route from Caddy
func (m *Manager) RemoveRoute(ctx context.Context, domain string) error {
	m.log.Info("removing route from caddy", "domain", domain)

	// Get all routes and find the one to remove
	routes, err := m.getRoutes(ctx)
	if err != nil {
		return err
	}

	for i, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				if host == domain {
					// Found the route, delete it
					url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes/%d", m.adminAPI, i)
					req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
					if err != nil {
						return err
					}

					resp, err := m.client.Do(req)
					if err != nil {
						return fmt.Errorf("failed to remove route: %w", err)
					}
					resp.Body.Close()

					if resp.StatusCode >= 400 {
						return fmt.Errorf("caddy returned error %d", resp.StatusCode)
					}

					m.log.Info("route removed successfully", "domain", domain)
					return nil
				}
			}
		}
	}

	return nil // Route not found, consider it removed
}

// GetRoute retrieves a route by domain
func (m *Manager) GetRoute(ctx context.Context, domain string) (*proxy.Route, error) {
	routes, err := m.getRoutes(ctx)
	if err != nil {
		return nil, err
	}

	for _, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				if host == domain {
					// Found the route
					var upstream *proxy.Upstream
					if len(route.Handle) > 0 && len(route.Handle[0].Upstreams) > 0 {
						dial := route.Handle[0].Upstreams[0].Dial
						var host string
						var port int
						fmt.Sscanf(dial, "%s:%d", &host, &port)
						upstream = &proxy.Upstream{Host: host, Port: port}
					}

					return &proxy.Route{
						Domain:     domain,
						BlueTarget: upstream, // Simplified
						ActiveSlot: proxy.SlotBlue,
					}, nil
				}
			}
		}
	}

	return nil, nil
}

// ListRoutes returns all routes
func (m *Manager) ListRoutes(ctx context.Context) ([]proxy.Route, error) {
	routes, err := m.getRoutes(ctx)
	if err != nil {
		return nil, err
	}

	var result []proxy.Route
	for _, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				var upstream *proxy.Upstream
				if len(route.Handle) > 0 && len(route.Handle[0].Upstreams) > 0 {
					dial := route.Handle[0].Upstreams[0].Dial
					var h string
					var p int
					fmt.Sscanf(dial, "%s:%d", &h, &p)
					upstream = &proxy.Upstream{Host: h, Port: p}
				}

				result = append(result, proxy.Route{
					Domain:     host,
					BlueTarget: upstream,
					ActiveSlot: proxy.SlotBlue,
				})
			}
		}
	}

	return result, nil
}

// SwitchTraffic switches traffic from one slot to another
func (m *Manager) SwitchTraffic(ctx context.Context, domain string, targetSlot proxy.Slot) error {
	m.log.Info("switching traffic",
		"domain", domain,
		"target_slot", targetSlot,
	)

	// This would update the upstream to point to the target slot's containers
	// For now, this is handled by UpdateRoute with the new upstream
	return nil
}

// ProvisionSSL provisions SSL for a domain
func (m *Manager) ProvisionSSL(ctx context.Context, domain string) error {
	// Caddy handles SSL automatically via Let's Encrypt
	// This is a no-op as Caddy auto-provisions certificates
	m.log.Info("SSL will be auto-provisioned by Caddy", "domain", domain)
	return nil
}

// HealthCheck checks if Caddy is healthy
func (m *Manager) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/config/", m.adminAPI)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("caddy health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("caddy returned status %d", resp.StatusCode)
	}

	return nil
}

// ReloadConfig reloads the Caddy configuration
func (m *Manager) ReloadConfig(ctx context.Context) error {
	url := fmt.Sprintf("%s/load", m.adminAPI)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reload caddy config: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// getRoutes retrieves all routes from Caddy
func (m *Manager) getRoutes(ctx context.Context) ([]CaddyRoute, error) {
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", m.adminAPI)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []CaddyRoute{}, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("caddy returned error %d", resp.StatusCode)
	}

	var routes []CaddyRoute
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// InitializeServer ensures the Caddy server configuration exists
func (m *Manager) InitializeServer(ctx context.Context) error {
	// Check if server exists
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0", m.adminAPI)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == 404 {
		// Create initial server configuration
		server := CaddyServer{
			Listen: []string{":80", ":443"},
			Routes: []CaddyRoute{},
		}

		serverJSON, err := json.Marshal(server)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(serverJSON))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := m.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to initialize server: %s", string(body))
		}
	}

	return nil
}
