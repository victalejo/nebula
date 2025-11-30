package service

import (
	"context"

	"github.com/google/uuid"

	apperrors "github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// DomainService handles domain business logic
type DomainService struct {
	store storage.Store
	log   logger.Logger
}

// NewDomainService creates a new domain service
func NewDomainService(store storage.Store, log logger.Logger) *DomainService {
	return &DomainService{
		store: store,
		log:   log,
	}
}

// CreateDomainRequest represents a request to create a domain
type CreateDomainRequest struct {
	Domain     string `json:"domain" binding:"required"`
	PathPrefix string `json:"path_prefix"` // "/" for root, "/api" for path-based routing
	SSLEnabled *bool  `json:"ssl_enabled"`
}

// DomainResponse represents a domain response
type DomainResponse struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	ServiceID  string `json:"service_id"`
	Domain     string `json:"domain"`
	PathPrefix string `json:"path_prefix"`
	ActiveSlot string `json:"active_slot"`
	SSLEnabled bool   `json:"ssl_enabled"`
	CreatedAt  string `json:"created_at"`
}

// Create creates a new domain for a service
func (s *DomainService) Create(ctx context.Context, projectID, serviceName string, req CreateDomainRequest) (*DomainResponse, error) {
	s.log.Info("creating domain", "project_id", projectID, "service", serviceName, "domain", req.Domain)

	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Resolve service
	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceName)
	}

	// Check if domain already exists
	existing, err := s.store.Domains().GetByDomain(ctx, req.Domain)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to check existing domain", err)
	}
	if existing != nil {
		return nil, apperrors.NewConflictError("domain already exists")
	}

	// Set defaults
	pathPrefix := req.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "/"
	}

	sslEnabled := true
	if req.SSLEnabled != nil {
		sslEnabled = *req.SSLEnabled
	}

	domain := &storage.Domain{
		ID:         uuid.New().String(),
		ProjectID:  project.ID,
		ServiceID:  service.ID,
		Domain:     req.Domain,
		PathPrefix: pathPrefix,
		ActiveSlot: "blue",
		SSLEnabled: sslEnabled,
	}

	if err := s.store.Domains().Create(ctx, domain); err != nil {
		return nil, apperrors.NewInternalError("failed to create domain", err)
	}

	s.log.Info("domain created", "id", domain.ID, "domain", domain.Domain, "service_id", service.ID)

	return s.toResponse(domain), nil
}

// Get retrieves a domain by domain name
func (s *DomainService) Get(ctx context.Context, domainName string) (*DomainResponse, error) {
	domain, err := s.store.Domains().GetByDomain(ctx, domainName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get domain", err)
	}
	if domain == nil {
		return nil, apperrors.NewNotFoundError("domain", domainName)
	}

	return s.toResponse(domain), nil
}

// GetByID retrieves a domain by ID
func (s *DomainService) GetByID(ctx context.Context, domainID string) (*DomainResponse, error) {
	domain, err := s.store.Domains().GetByID(ctx, domainID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get domain", err)
	}
	if domain == nil {
		return nil, apperrors.NewNotFoundError("domain", domainID)
	}

	return s.toResponse(domain), nil
}

// ListByProject returns all domains for a project
func (s *DomainService) ListByProject(ctx context.Context, projectID string) ([]*DomainResponse, error) {
	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	domains, err := s.store.Domains().ListByProjectID(ctx, project.ID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list domains", err)
	}

	responses := make([]*DomainResponse, len(domains))
	for i, d := range domains {
		responses[i] = s.toResponse(d)
	}

	return responses, nil
}

// ListByService returns all domains for a service
func (s *DomainService) ListByService(ctx context.Context, projectID, serviceName string) ([]*DomainResponse, error) {
	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Resolve service
	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceName)
	}

	domains, err := s.store.Domains().ListByServiceID(ctx, service.ID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list domains", err)
	}

	responses := make([]*DomainResponse, len(domains))
	for i, d := range domains {
		responses[i] = s.toResponse(d)
	}

	return responses, nil
}

// UpdateDomainRequest represents a request to update a domain
type UpdateDomainRequest struct {
	PathPrefix *string `json:"path_prefix"`
	SSLEnabled *bool   `json:"ssl_enabled"`
}

// Update updates a domain
func (s *DomainService) Update(ctx context.Context, domainName string, req UpdateDomainRequest) (*DomainResponse, error) {
	domain, err := s.store.Domains().GetByDomain(ctx, domainName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get domain", err)
	}
	if domain == nil {
		return nil, apperrors.NewNotFoundError("domain", domainName)
	}

	// Apply updates
	if req.PathPrefix != nil {
		domain.PathPrefix = *req.PathPrefix
	}
	if req.SSLEnabled != nil {
		domain.SSLEnabled = *req.SSLEnabled
	}

	if err := s.store.Domains().Update(ctx, domain); err != nil {
		return nil, apperrors.NewInternalError("failed to update domain", err)
	}

	s.log.Info("domain updated", "id", domain.ID, "domain", domain.Domain)

	return s.toResponse(domain), nil
}

// UpdateActiveSlot updates the active slot for a domain
func (s *DomainService) UpdateActiveSlot(ctx context.Context, domainName, slot string) error {
	domain, err := s.store.Domains().GetByDomain(ctx, domainName)
	if err != nil {
		return apperrors.NewInternalError("failed to get domain", err)
	}
	if domain == nil {
		return apperrors.NewNotFoundError("domain", domainName)
	}

	domain.ActiveSlot = slot
	if err := s.store.Domains().Update(ctx, domain); err != nil {
		return apperrors.NewInternalError("failed to update domain slot", err)
	}

	return nil
}

// Delete deletes a domain
func (s *DomainService) Delete(ctx context.Context, domainName string) error {
	domain, err := s.store.Domains().GetByDomain(ctx, domainName)
	if err != nil {
		return apperrors.NewInternalError("failed to get domain", err)
	}
	if domain == nil {
		return apperrors.NewNotFoundError("domain", domainName)
	}

	if err := s.store.Domains().Delete(ctx, domain.ID); err != nil {
		return apperrors.NewInternalError("failed to delete domain", err)
	}

	s.log.Info("domain deleted", "id", domain.ID, "domain", domain.Domain)

	return nil
}

func (s *DomainService) resolveProject(ctx context.Context, projectIDOrName string) (*storage.Project, error) {
	project, err := s.store.Projects().GetByID(ctx, projectIDOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Projects().GetByName(ctx, projectIDOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", projectIDOrName)
	}
	return project, nil
}

func (s *DomainService) toResponse(domain *storage.Domain) *DomainResponse {
	return &DomainResponse{
		ID:         domain.ID,
		ProjectID:  domain.ProjectID,
		ServiceID:  domain.ServiceID,
		Domain:     domain.Domain,
		PathPrefix: domain.PathPrefix,
		ActiveSlot: domain.ActiveSlot,
		SSLEnabled: domain.SSLEnabled,
		CreatedAt:  domain.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
