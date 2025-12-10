package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/config"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
	"github.com/victalejo/nebula/internal/version"
)

const (
	githubAPIURL   = "https://api.github.com/repos/victalejo/nebula/releases/latest"
	githubOwner    = "victalejo"
	githubRepo     = "nebula"
	maxBackups     = 3
	backupDir      = "./data/backups"
	updateStateFile = ".nebula-update"
)

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	Available      bool      `json:"available"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	ReleaseNotes   string    `json:"release_notes,omitempty"`
	DownloadURL    string    `json:"download_url,omitempty"`
	ChecksumURL    string    `json:"checksum_url,omitempty"`
	PublishedAt    time.Time `json:"published_at,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

// UpdateStatus represents the current update operation status
type UpdateStatus struct {
	State    string  `json:"state"` // "idle", "checking", "downloading", "ready", "applying", "restarting"
	Progress float64 `json:"progress"`
	Error    string  `json:"error,omitempty"`
}

// BackupInfo represents a binary backup
type BackupInfo struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Path      string    `json:"path"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// UpdateService handles auto-update functionality
type UpdateService struct {
	config     config.UpdateConfig
	store      storage.Store
	log        logger.Logger
	httpClient *http.Client

	mu         sync.RWMutex
	lastCheck  *UpdateInfo
	status     UpdateStatus
	downloadedPath string
}

// NewUpdateService creates a new update service
func NewUpdateService(cfg config.UpdateConfig, store storage.Store, log logger.Logger) *UpdateService {
	return &UpdateService{
		config:     cfg,
		store:      store,
		log:        log,
		status:     UpdateStatus{State: "idle"},
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// GetConfig returns the current update configuration
func (s *UpdateService) GetConfig() config.UpdateConfig {
	return s.config
}

// UpdateConfig updates the configuration
func (s *UpdateService) UpdateConfig(cfg config.UpdateConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
}

// CheckForUpdates queries GitHub for new releases
func (s *UpdateService) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	s.setStatus("checking", 0, "")
	defer func() {
		if s.status.State == "checking" {
			s.setStatus("idle", 0, "")
		}
	}()

	s.log.Info("checking for updates", "current_version", version.Version)

	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIURL, nil)
	if err != nil {
		s.setStatus("idle", 0, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Nebula-PaaS/"+version.Version)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.setStatus("idle", 0, err.Error())
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.setStatus("idle", 0, fmt.Sprintf("GitHub API returned %d", resp.StatusCode))
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		s.setStatus("idle", 0, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(version.Version, "v")

	info := &UpdateInfo{
		CurrentVersion: version.Version,
		LatestVersion:  latestVersion,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
		CheckedAt:      time.Now(),
	}

	// Compare versions
	info.Available = s.isNewerVersion(latestVersion, currentVersion)

	if info.Available {
		// Find the correct asset for this platform
		assetName := fmt.Sprintf("nebula-server-%s-%s", runtime.GOOS, runtime.GOARCH)
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, assetName) && !strings.HasSuffix(asset.Name, ".txt") {
				info.DownloadURL = asset.BrowserDownloadURL
			}
			if asset.Name == "checksums.txt" {
				info.ChecksumURL = asset.BrowserDownloadURL
			}
		}

		if info.DownloadURL == "" {
			s.log.Warn("no compatible binary found for platform", "os", runtime.GOOS, "arch", runtime.GOARCH)
		}
	}

	s.mu.Lock()
	s.lastCheck = info
	s.mu.Unlock()

	s.log.Info("update check completed",
		"available", info.Available,
		"current", info.CurrentVersion,
		"latest", info.LatestVersion,
	)

	return info, nil
}

// DownloadAndApply downloads and applies the update
func (s *UpdateService) DownloadAndApply(ctx context.Context) error {
	s.mu.RLock()
	info := s.lastCheck
	s.mu.RUnlock()

	if info == nil || !info.Available {
		return fmt.Errorf("no update available")
	}

	if info.DownloadURL == "" {
		return fmt.Errorf("no download URL available for this platform")
	}

	s.log.Info("starting update download", "version", info.LatestVersion, "url", info.DownloadURL)

	// Create backup first
	backup, err := s.createBackup()
	if err != nil {
		s.log.Error("failed to create backup", "error", err)
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to create backup: %w", err)
	}
	s.log.Info("backup created", "id", backup.ID, "path", backup.Path)

	// Download update
	s.setStatus("downloading", 0, "")
	tempFile, err := s.downloadUpdate(ctx, info.DownloadURL)
	if err != nil {
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Verify checksum if available
	if info.ChecksumURL != "" {
		if err := s.verifyChecksum(ctx, tempFile, info.ChecksumURL); err != nil {
			os.Remove(tempFile)
			s.setStatus("idle", 0, err.Error())
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		s.log.Info("checksum verified")
	}

	s.mu.Lock()
	s.downloadedPath = tempFile
	s.mu.Unlock()

	s.setStatus("ready", 100, "")
	s.log.Info("update ready to apply", "path", tempFile)

	// If auto mode, apply immediately
	if s.config.Mode == "auto" {
		return s.ApplyUpdate(ctx)
	}

	return nil
}

// ApplyUpdate replaces the binary and restarts
func (s *UpdateService) ApplyUpdate(ctx context.Context) error {
	s.mu.RLock()
	downloadedPath := s.downloadedPath
	s.mu.RUnlock()

	if downloadedPath == "" {
		return fmt.Errorf("no update downloaded")
	}

	s.setStatus("applying", 0, "")
	s.log.Info("applying update", "source", downloadedPath)

	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Replace binary atomically
	newPath := executable + ".new"
	if err := copyFile(downloadedPath, newPath); err != nil {
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to copy new binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(newPath, 0755); err != nil {
		os.Remove(newPath)
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(newPath, executable); err != nil {
		os.Remove(newPath)
		s.setStatus("idle", 0, err.Error())
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Clean up temp file
	os.Remove(downloadedPath)

	s.log.Info("binary replaced, restarting...")
	s.setStatus("restarting", 100, "")

	// Save restart state
	s.saveRestartState()

	// Restart the process
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.restart(executable)
	}()

	return nil
}

// Rollback restores a previous version
func (s *UpdateService) Rollback(ctx context.Context, backupID string) error {
	backup, err := s.store.BinaryBackups().Get(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}
	if backup == nil {
		return fmt.Errorf("backup not found")
	}

	s.log.Info("rolling back to version", "version", backup.Version, "backup_id", backupID)

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Copy backup to new path
	newPath := executable + ".new"
	if err := copyFile(backup.BinaryPath, newPath); err != nil {
		return fmt.Errorf("failed to copy backup: %w", err)
	}

	if err := os.Chmod(newPath, 0755); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(newPath, executable); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	s.log.Info("rollback completed, restarting...")

	go func() {
		time.Sleep(500 * time.Millisecond)
		s.restart(executable)
	}()

	return nil
}

// ListBackups returns all available backups
func (s *UpdateService) ListBackups(ctx context.Context) ([]*storage.BinaryBackup, error) {
	return s.store.BinaryBackups().List(ctx)
}

// GetStatus returns the current update status
func (s *UpdateService) GetStatus() UpdateStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// GetLastCheck returns the last update check info
func (s *UpdateService) GetLastCheck() *UpdateInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastCheck
}

// StartBackgroundChecker starts periodic update checks
func (s *UpdateService) StartBackgroundChecker(ctx context.Context) {
	if s.config.Mode == "disabled" {
		s.log.Info("auto-update disabled")
		return
	}

	interval := time.Duration(s.config.CheckInterval) * time.Minute
	s.log.Info("starting background update checker", "interval", interval, "mode", s.config.Mode)

	// Initial check after 1 minute
	time.AfterFunc(time.Minute, func() {
		info, err := s.CheckForUpdates(ctx)
		if err != nil {
			s.log.Error("background update check failed", "error", err)
			return
		}
		if info.Available && s.config.Mode == "auto" {
			s.log.Info("auto-update: new version available, downloading...")
			if err := s.DownloadAndApply(ctx); err != nil {
				s.log.Error("auto-update failed", "error", err)
			}
		}
	})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := s.CheckForUpdates(ctx)
			if err != nil {
				s.log.Error("background update check failed", "error", err)
				continue
			}
			if info.Available && s.config.Mode == "auto" {
				s.log.Info("auto-update: new version available, downloading...")
				if err := s.DownloadAndApply(ctx); err != nil {
					s.log.Error("auto-update failed", "error", err)
				}
			}
		}
	}
}

// Internal methods

func (s *UpdateService) setStatus(state string, progress float64, errMsg string) {
	s.mu.Lock()
	s.status = UpdateStatus{State: state, Progress: progress, Error: errMsg}
	s.mu.Unlock()
}

func (s *UpdateService) isNewerVersion(latest, current string) bool {
	// Skip if dev version
	if current == "dev" || current == "" {
		return false
	}

	// Simple comparison - in production use semver library
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	return latest != current && latest > current
}

func (s *UpdateService) downloadUpdate(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Nebula-PaaS/"+version.Version)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tempFile := filepath.Join(os.TempDir(), "nebula-update-"+uuid.New().String())
	f, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Download with progress
	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				os.Remove(tempFile)
				return "", writeErr
			}
			downloaded += int64(n)
			if totalSize > 0 {
				progress := float64(downloaded) / float64(totalSize) * 100
				s.setStatus("downloading", progress, "")
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			os.Remove(tempFile)
			return "", readErr
		}
	}

	return tempFile, nil
}

func (s *UpdateService) verifyChecksum(ctx context.Context, filePath, checksumURL string) error {
	// Download checksums file
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse checksums (format: hash  filename)
	expectedHash := ""
	assetName := fmt.Sprintf("nebula-server-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.Contains(parts[1], assetName) {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	// Calculate file hash
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func (s *UpdateService) createBackup() (*BackupInfo, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, err
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, err
	}

	// Calculate hash of current binary
	f, err := os.Open(executable)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()
	hash := hex.EncodeToString(h.Sum(nil))

	// Create backup file
	backupID := uuid.New().String()
	backupPath := filepath.Join(backupDir, fmt.Sprintf("nebula-%s-%s", version.Version, backupID[:8]))

	if err := copyFile(executable, backupPath); err != nil {
		return nil, err
	}

	backup := &storage.BinaryBackup{
		ID:         backupID,
		Version:    version.Version,
		BinaryPath: backupPath,
		BinaryHash: hash,
		CreatedAt:  time.Now(),
	}

	// Save to database
	if err := s.store.BinaryBackups().Create(context.Background(), backup); err != nil {
		os.Remove(backupPath)
		return nil, err
	}

	// Clean old backups
	s.cleanOldBackups()

	return &BackupInfo{
		ID:        backup.ID,
		Version:   backup.Version,
		Path:      backup.BinaryPath,
		Hash:      backup.BinaryHash,
		CreatedAt: backup.CreatedAt,
	}, nil
}

func (s *UpdateService) cleanOldBackups() {
	ctx := context.Background()
	backups, err := s.store.BinaryBackups().List(ctx)
	if err != nil {
		s.log.Error("failed to list backups for cleanup", "error", err)
		return
	}

	if len(backups) <= maxBackups {
		return
	}

	// Delete oldest backups
	for i := maxBackups; i < len(backups); i++ {
		if err := os.Remove(backups[i].BinaryPath); err != nil {
			s.log.Warn("failed to remove old backup file", "path", backups[i].BinaryPath, "error", err)
		}
		if err := s.store.BinaryBackups().Delete(ctx, backups[i].ID); err != nil {
			s.log.Warn("failed to delete backup record", "id", backups[i].ID, "error", err)
		}
	}
}

func (s *UpdateService) saveRestartState() {
	executable, _ := os.Executable()
	dir := filepath.Dir(executable)
	stateFile := filepath.Join(dir, updateStateFile)
	_ = os.WriteFile(stateFile, []byte("pending"), 0644)
}

func (s *UpdateService) restart(executable string) {
	s.log.Info("restarting process...")

	// Check if running as systemd service
	if os.Getenv("INVOCATION_ID") != "" || s.isSystemdService() {
		s.log.Info("detected systemd service, using systemctl restart")
		cmd := exec.Command("systemctl", "restart", "nebula")
		if err := cmd.Run(); err != nil {
			s.log.Error("systemctl restart failed", "error", err)
			// Fallback to exit - systemd will restart the service
			os.Exit(0)
		}
		return
	}

	// Try syscall.Exec first (replaces current process)
	args := os.Args
	env := os.Environ()

	err := syscall.Exec(executable, args, env)
	if err != nil {
		s.log.Warn("syscall.Exec failed, trying exec.Command", "error", err)
		// Fallback: start new process and exit
		cmd := exec.Command(executable, args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		if err := cmd.Start(); err != nil {
			s.log.Error("failed to restart", "error", err)
			return
		}
		os.Exit(0)
	}
}

func (s *UpdateService) isSystemdService() bool {
	// Check if systemd is managing this process
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		// Check if nebula service exists
		cmd := exec.Command("systemctl", "is-active", "nebula")
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
