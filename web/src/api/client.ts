const API_BASE = '/api/v1';

interface ApiError {
  error: string;
  code?: string;
}

interface ApiResponse<T> {
  data: T;
}

class ApiClient {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('nebula_token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('nebula_token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('nebula_token');
  }

  getToken(): string | null {
    return this.token;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        error: response.statusText,
      }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  }

  async get<T>(path: string): Promise<T> {
    return this.request<T>('GET', path);
  }

  async post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>('POST', path, body);
  }

  async put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>('PUT', path, body);
  }

  async delete<T>(path: string): Promise<T> {
    return this.request<T>('DELETE', path);
  }

  // Auth
  async login(username: string, password: string): Promise<{ token: string }> {
    const result = await this.post<ApiResponse<{ token: string }>>('/auth/login', {
      username,
      password,
    });
    this.setToken(result.data.token);
    return result.data;
  }

  async getUser(): Promise<{ username: string }> {
    const result = await this.get<ApiResponse<{ username: string }>>('/auth/me');
    return result.data;
  }

  // Apps
  async listApps(): Promise<App[]> {
    const result = await this.get<ApiResponse<App[]>>('/apps');
    return result.data;
  }

  async getApp(name: string): Promise<App> {
    const result = await this.get<ApiResponse<App>>(`/apps/${name}`);
    return result.data;
  }

  async createApp(data: CreateAppRequest): Promise<App> {
    const result = await this.post<ApiResponse<App>>('/apps', data);
    return result.data;
  }

  async deleteApp(name: string): Promise<void> {
    await this.delete(`/apps/${name}`);
  }

  // Deployments
  async listDeployments(appName: string): Promise<Deployment[]> {
    const result = await this.get<ApiResponse<Deployment[]>>(
      `/apps/${appName}/deployments`
    );
    return result.data;
  }

  async deployImage(appName: string, data: DeployImageRequest): Promise<Deployment> {
    const result = await this.post<ApiResponse<Deployment>>(
      `/apps/${appName}/deploy/image`,
      data
    );
    return result.data;
  }

  // Logs - returns EventSource for SSE
  streamLogs(
    appName: string,
    options: { follow?: boolean; tail?: number; service?: string } = {}
  ): EventSource {
    const params = new URLSearchParams();
    if (options.follow) params.set('follow', 'true');
    if (options.tail) params.set('tail', options.tail.toString());
    if (options.service) params.set('service', options.service);

    const url = `${API_BASE}/apps/${appName}/logs?${params.toString()}`;
    const eventSource = new EventSource(url);
    return eventSource;
  }
}

// Types
export interface App {
  id: string;
  name: string;
  deployment_mode: 'git' | 'docker_image' | 'docker_compose';
  domain: string;
  git_repo?: string;
  git_branch?: string;
  docker_image?: string;
  compose_file?: string;
  env_vars: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface Deployment {
  id: string;
  app_id: string;
  version: string;
  slot: 'blue' | 'green';
  status: 'pending' | 'preparing' | 'deploying' | 'running' | 'stopped' | 'failed';
  container_ids: string[];
  created_at: string;
  finished_at?: string;
}

export interface CreateAppRequest {
  name: string;
  deployment_mode: 'git' | 'docker_image' | 'docker_compose';
  domain?: string;
  git_repo?: string;
  git_branch?: string;
  docker_image?: string;
  compose_file?: string;
  env_vars?: Record<string, string>;
}

export interface DeployImageRequest {
  image: string;
  tag?: string;
  env_vars?: Record<string, string>;
}

export const api = new ApiClient();
export default api;
