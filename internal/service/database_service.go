package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// DatabaseType represents supported database types
type DatabaseType string

const (
	DBTypePostgres DatabaseType = "postgres"
	DBTypeMySQL    DatabaseType = "mysql"
	DBTypeRedis    DatabaseType = "redis"
	DBTypeMongoDB  DatabaseType = "mongodb"
)

// DatabaseConfig holds configuration for each database type
type DatabaseConfig struct {
	Image       string
	DefaultPort int
	EnvUser     string
	EnvPassword string
	EnvDatabase string
	HealthCmd   []string
}

var databaseConfigs = map[DatabaseType]DatabaseConfig{
	DBTypePostgres: {
		Image:       "postgres:16-alpine",
		DefaultPort: 5432,
		EnvUser:     "POSTGRES_USER",
		EnvPassword: "POSTGRES_PASSWORD",
		EnvDatabase: "POSTGRES_DB",
		HealthCmd:   []string{"CMD-SHELL", "pg_isready -U $POSTGRES_USER -d $POSTGRES_DB"},
	},
	DBTypeMySQL: {
		Image:       "mysql:8",
		DefaultPort: 3306,
		EnvUser:     "MYSQL_USER",
		EnvPassword: "MYSQL_PASSWORD",
		EnvDatabase: "MYSQL_DATABASE",
		HealthCmd:   []string{"CMD", "mysqladmin", "ping", "-h", "localhost"},
	},
	DBTypeRedis: {
		Image:       "redis:7-alpine",
		DefaultPort: 6379,
		EnvPassword: "REDIS_PASSWORD",
		HealthCmd:   []string{"CMD", "redis-cli", "ping"},
	},
	DBTypeMongoDB: {
		Image:       "mongo:7",
		DefaultPort: 27017,
		EnvUser:     "MONGO_INITDB_ROOT_USERNAME",
		EnvPassword: "MONGO_INITDB_ROOT_PASSWORD",
		EnvDatabase: "MONGO_INITDB_DATABASE",
		HealthCmd:   []string{"CMD", "mongosh", "--eval", "db.adminCommand('ping')"},
	},
}

type DatabaseService struct {
	dbRepo    storage.DatabaseRepository
	backupRepo storage.BackupRepository
	runtime   container.Runtime
	log       logger.Logger
	dataDir   string
}

func NewDatabaseService(
	dbRepo storage.DatabaseRepository,
	backupRepo storage.BackupRepository,
	runtime container.Runtime,
	log logger.Logger,
	dataDir string,
) *DatabaseService {
	return &DatabaseService{
		dbRepo:    dbRepo,
		backupRepo: backupRepo,
		runtime:   runtime,
		log:       log,
		dataDir:   dataDir,
	}
}

type CreateDatabaseInput struct {
	Name     string
	Type     DatabaseType
	Version  string // Optional, e.g., "16" for postgres:16
	AppID    string // Optional, link to an app
}

type DatabaseInfo struct {
	ID            string
	Name          string
	Type          DatabaseType
	Host          string
	Port          int
	Username      string
	Password      string
	Database      string
	ConnectionURL string
	Status        string
	CreatedAt     time.Time
}

func (s *DatabaseService) Create(ctx context.Context, input CreateDatabaseInput) (*DatabaseInfo, error) {
	// Validate database type
	config, ok := databaseConfigs[input.Type]
	if !ok {
		return nil, errors.New(errors.ErrTypeValidation, fmt.Sprintf("unsupported database type: %s", input.Type))
	}

	// Validate name
	if input.Name == "" {
		return nil, errors.New(errors.ErrTypeValidation, "database name is required")
	}
	if !isValidName(input.Name) {
		return nil, errors.New(errors.ErrTypeValidation, "invalid database name")
	}

	// Check if database already exists
	existing, _ := s.dbRepo.GetByName(ctx, input.Name)
	if existing != nil {
		return nil, errors.New(errors.ErrTypeConflict, "database already exists")
	}

	// Generate credentials
	password := generatePassword(24)
	username := input.Name
	dbName := input.Name

	// Determine image
	image := config.Image
	if input.Version != "" {
		parts := strings.Split(config.Image, ":")
		image = fmt.Sprintf("%s:%s", parts[0], input.Version)
	}

	// Create database record
	db := &storage.Database{
		ID:        uuid.New().String(),
		Name:      input.Name,
		Type:      string(input.Type),
		Host:      "localhost",
		Port:      config.DefaultPort,
		Username:  username,
		Password:  password,
		Database:  dbName,
		Status:    "creating",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if input.AppID != "" {
		db.AppID = &input.AppID
	}

	if err := s.dbRepo.Create(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create database record: %w", err)
	}

	// Build container config
	containerName := fmt.Sprintf("nebula-db-%s", input.Name)
	volumeName := fmt.Sprintf("nebula-db-%s-data", input.Name)

	var env []string
	if config.EnvUser != "" {
		env = append(env, fmt.Sprintf("%s=%s", config.EnvUser, username))
	}
	if config.EnvPassword != "" {
		env = append(env, fmt.Sprintf("%s=%s", config.EnvPassword, password))
	}
	if config.EnvDatabase != "" {
		env = append(env, fmt.Sprintf("%s=%s", config.EnvDatabase, dbName))
	}

	// MySQL requires root password
	if input.Type == DBTypeMySQL {
		env = append(env, fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", password))
	}

	containerConfig := &container.ContainerConfig{
		Name:  containerName,
		Image: image,
		Env:   env,
		Ports: []container.PortMapping{
			{HostPort: 0, ContainerPort: config.DefaultPort},
		},
		Volumes: []container.VolumeMount{
			{Source: volumeName, Target: getDataPath(input.Type)},
		},
		Labels: map[string]string{
			"nebula.database":      "true",
			"nebula.database.name": input.Name,
			"nebula.database.type": string(input.Type),
		},
		RestartPolicy: "unless-stopped",
		HealthCheck: &container.HealthCheck{
			Test:     config.HealthCmd,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  5,
		},
	}

	// Pull image
	s.log.Info("pulling database image", "image", image)
	if err := s.runtime.PullImage(ctx, image, nil); err != nil {
		s.log.Warn("failed to pull image, trying local", "error", err)
	}

	// Create container
	s.log.Info("creating database container", "name", containerName)
	containerID, err := s.runtime.CreateContainer(ctx, containerConfig)
	if err != nil {
		db.Status = "failed"
		_ = s.dbRepo.Update(ctx, db)
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	db.ContainerID = containerID

	// Start container
	if err := s.runtime.StartContainer(ctx, containerID); err != nil {
		_ = s.runtime.RemoveContainer(ctx, containerID)
		db.Status = "failed"
		_ = s.dbRepo.Update(ctx, db)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get assigned port
	info, err := s.runtime.InspectContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	for _, pm := range info.Ports {
		if pm.ContainerPort == config.DefaultPort && pm.HostPort > 0 {
			db.Port = pm.HostPort
			break
		}
	}

	db.Status = "running"
	if err := s.dbRepo.Update(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to update database record: %w", err)
	}

	return s.toInfo(db), nil
}

func (s *DatabaseService) Get(ctx context.Context, name string) (*DatabaseInfo, error) {
	db, err := s.dbRepo.GetByName(ctx, name)
	if err != nil {
		return nil, errors.New(errors.ErrTypeNotFound, "database not found")
	}
	return s.toInfo(db), nil
}

func (s *DatabaseService) List(ctx context.Context) ([]*DatabaseInfo, error) {
	dbs, err := s.dbRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*DatabaseInfo, len(dbs))
	for i, db := range dbs {
		result[i] = s.toInfo(db)
	}
	return result, nil
}

func (s *DatabaseService) Delete(ctx context.Context, name string) error {
	db, err := s.dbRepo.GetByName(ctx, name)
	if err != nil {
		return errors.New(errors.ErrTypeNotFound, "database not found")
	}

	// Stop and remove container
	if db.ContainerID != "" {
		_ = s.runtime.StopContainer(ctx, db.ContainerID, 30*time.Second)
		_ = s.runtime.RemoveContainer(ctx, db.ContainerID)
	}

	// Note: We keep the volume for safety. User can manually delete it.

	return s.dbRepo.Delete(ctx, db.ID)
}

func (s *DatabaseService) GetStatus(ctx context.Context, name string) (string, error) {
	db, err := s.dbRepo.GetByName(ctx, name)
	if err != nil {
		return "", errors.New(errors.ErrTypeNotFound, "database not found")
	}

	if db.ContainerID == "" {
		return "unknown", nil
	}

	info, err := s.runtime.InspectContainer(ctx, db.ContainerID)
	if err != nil {
		return "stopped", nil
	}

	if info.State != "running" {
		return info.State, nil
	}

	if info.Health == "healthy" || info.Health == "" {
		return "running", nil
	}

	return info.Health, nil
}

func (s *DatabaseService) Restart(ctx context.Context, name string) error {
	db, err := s.dbRepo.GetByName(ctx, name)
	if err != nil {
		return errors.New(errors.ErrTypeNotFound, "database not found")
	}

	if db.ContainerID == "" {
		return errors.New(errors.ErrTypeValidation, "database has no container")
	}

	return s.runtime.RestartContainer(ctx, db.ContainerID, 30*time.Second)
}

func (s *DatabaseService) toInfo(db *storage.Database) *DatabaseInfo {
	info := &DatabaseInfo{
		ID:        db.ID,
		Name:      db.Name,
		Type:      DatabaseType(db.Type),
		Host:      db.Host,
		Port:      db.Port,
		Username:  db.Username,
		Password:  db.Password,
		Database:  db.Database,
		Status:    db.Status,
		CreatedAt: db.CreatedAt,
	}

	// Build connection URL
	switch DatabaseType(db.Type) {
	case DBTypePostgres:
		info.ConnectionURL = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
			db.Username, db.Password, db.Host, db.Port, db.Database)
	case DBTypeMySQL:
		info.ConnectionURL = fmt.Sprintf("mysql://%s:%s@%s:%d/%s",
			db.Username, db.Password, db.Host, db.Port, db.Database)
	case DBTypeRedis:
		if db.Password != "" {
			info.ConnectionURL = fmt.Sprintf("redis://:%s@%s:%d",
				db.Password, db.Host, db.Port)
		} else {
			info.ConnectionURL = fmt.Sprintf("redis://%s:%d", db.Host, db.Port)
		}
	case DBTypeMongoDB:
		info.ConnectionURL = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin",
			db.Username, db.Password, db.Host, db.Port, db.Database)
	}

	return info
}

func getDataPath(dbType DatabaseType) string {
	switch dbType {
	case DBTypePostgres:
		return "/var/lib/postgresql/data"
	case DBTypeMySQL:
		return "/var/lib/mysql"
	case DBTypeRedis:
		return "/data"
	case DBTypeMongoDB:
		return "/data/db"
	default:
		return "/data"
	}
}

func generatePassword(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func isValidName(name string) bool {
	if len(name) < 1 || len(name) > 63 {
		return false
	}
	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '-' && i > 0)) {
			return false
		}
	}
	return name[len(name)-1] != '-'
}
