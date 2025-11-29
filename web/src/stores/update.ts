import { createSignal, createRoot } from 'solid-js';

// Types
export interface SystemInfo {
  version: string;
  build_time: string;
  commit: string;
  update_mode: 'auto' | 'notify' | 'disabled';
  check_interval: number;
}

export interface UpdateInfo {
  available: boolean;
  current_version: string;
  latest_version?: string;
  release_notes?: string;
  download_url?: string;
  published_at?: string;
  checked_at: string;
}

export interface UpdateStatus {
  state: 'idle' | 'checking' | 'downloading' | 'ready' | 'applying' | 'restarting';
  progress: number;
  error?: string;
}

export interface UpdateConfig {
  mode: 'auto' | 'notify' | 'disabled';
  check_interval: number;
}

export interface BinaryBackup {
  id: string;
  version: string;
  binary_path: string;
  binary_hash: string;
  created_at: string;
}

const API_BASE = '/api/v1';

async function fetchWithAuth<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('nebula_token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
}

function createUpdateStore() {
  const [systemInfo, setSystemInfo] = createSignal<SystemInfo | null>(null);
  const [updateInfo, setUpdateInfo] = createSignal<UpdateInfo | null>(null);
  const [status, setStatus] = createSignal<UpdateStatus>({ state: 'idle', progress: 0 });
  const [backups, setBackups] = createSignal<BinaryBackup[]>([]);
  const [dismissed, setDismissed] = createSignal(false);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  async function fetchSystemInfo() {
    try {
      const result = await fetchWithAuth<{ data: SystemInfo }>('/system/info');
      setSystemInfo(result.data);
    } catch (e) {
      console.error('Failed to fetch system info:', e);
    }
  }

  async function fetchUpdateStatus() {
    try {
      const result = await fetchWithAuth<{ data: { status: UpdateStatus; last_check: UpdateInfo | null } }>('/system/updates');
      setStatus(result.data.status);
      if (result.data.last_check) {
        setUpdateInfo(result.data.last_check);
      }
    } catch (e) {
      console.error('Failed to fetch update status:', e);
    }
  }

  async function checkForUpdates() {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchWithAuth<{ data: UpdateInfo }>('/system/updates/check', { method: 'POST' });
      setUpdateInfo(result.data);
      setDismissed(false);
    } catch (e: any) {
      setError(e.message);
      console.error('Failed to check for updates:', e);
    } finally {
      setLoading(false);
    }
  }

  async function applyUpdate() {
    setLoading(true);
    setError(null);
    try {
      await fetchWithAuth<{ message: string }>('/system/updates/apply', { method: 'POST' });
      // Start polling for status
      const pollInterval = setInterval(async () => {
        await fetchUpdateStatus();
        const currentStatus = status();
        if (currentStatus.state === 'ready' || currentStatus.state === 'restarting' || currentStatus.error) {
          clearInterval(pollInterval);
          setLoading(false);
        }
      }, 1000);
    } catch (e: any) {
      setError(e.message);
      setLoading(false);
      console.error('Failed to apply update:', e);
    }
  }

  async function updateConfig(config: Partial<UpdateConfig>) {
    try {
      const result = await fetchWithAuth<{ data: UpdateConfig }>('/system/updates/config', {
        method: 'PUT',
        body: JSON.stringify(config),
      });
      // Update system info with new config
      const info = systemInfo();
      if (info) {
        setSystemInfo({
          ...info,
          update_mode: result.data.mode,
          check_interval: result.data.check_interval,
        });
      }
    } catch (e: any) {
      setError(e.message);
      console.error('Failed to update config:', e);
    }
  }

  async function fetchBackups() {
    try {
      const result = await fetchWithAuth<{ data: BinaryBackup[] }>('/system/backups');
      setBackups(result.data || []);
    } catch (e) {
      console.error('Failed to fetch backups:', e);
    }
  }

  async function rollback(backupId: string) {
    setLoading(true);
    setError(null);
    try {
      await fetchWithAuth<{ message: string }>(`/system/rollback/${backupId}`, { method: 'POST' });
    } catch (e: any) {
      setError(e.message);
      console.error('Failed to rollback:', e);
    } finally {
      setLoading(false);
    }
  }

  function dismiss() {
    setDismissed(true);
  }

  function clearError() {
    setError(null);
  }

  return {
    // State
    systemInfo,
    updateInfo,
    status,
    backups,
    dismissed,
    loading,
    error,
    // Actions
    fetchSystemInfo,
    fetchUpdateStatus,
    checkForUpdates,
    applyUpdate,
    updateConfig,
    fetchBackups,
    rollback,
    dismiss,
    clearError,
  };
}

export const updateStore = createRoot(createUpdateStore);
