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
    const result = await this.post<{ token: string; expires_at: string }>('/auth/login', {
      username,
      password,
    });
    this.setToken(result.token);
    return result;
  }

  async getUser(): Promise<{ username: string }> {
    return this.get<{ username: string }>('/auth/me');
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

  async deployGit(appName: string, data: DeployGitRequest): Promise<Deployment> {
    const result = await this.post<ApiResponse<Deployment>>(
      `/apps/${appName}/deploy/git`,
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
    // EventSource doesn't support Authorization headers, so pass token as query param
    if (this.token) params.set('token', this.token);

    const url = `${API_BASE}/apps/${appName}/logs?${params.toString()}`;
    const eventSource = new EventSource(url);
    return eventSource;
  }

  // Deployment Logs - returns EventSource for SSE
  streamDeploymentLogs(
    appName: string,
    deploymentId: string,
    options: { follow?: boolean; tail?: number } = {}
  ): EventSource {
    const params = new URLSearchParams();
    if (options.follow) params.set('follow', 'true');
    if (options.tail) params.set('tail', options.tail.toString());
    if (this.token) params.set('token', this.token);

    const url = `${API_BASE}/apps/${appName}/deployments/${deploymentId}/logs?${params.toString()}`;
    const eventSource = new EventSource(url);
    return eventSource;
  }

  // Status Streaming - returns EventSource for SSE (real-time status updates)
  streamProjectStatus(projectId: string): EventSource {
    const params = new URLSearchParams();
    if (this.token) params.set('token', this.token);

    const url = `${API_BASE}/projects/${projectId}/status/stream?${params.toString()}`;
    return new EventSource(url);
  }

  // Global Status Streaming - returns EventSource for SSE (all projects)
  streamGlobalStatus(): EventSource {
    const params = new URLSearchParams();
    if (this.token) params.set('token', this.token);

    const url = `${API_BASE}/status/stream?${params.toString()}`;
    return new EventSource(url);
  }

  // Settings
  async getGitHubTokenStatus(): Promise<GitHubTokenStatus> {
    return this.get<GitHubTokenStatus>('/settings/github-token');
  }

  async setGitHubToken(token: string): Promise<{ message: string }> {
    return this.put<{ message: string }>('/settings/github-token', { token });
  }

  async deleteGitHubToken(): Promise<{ message: string }> {
    return this.delete<{ message: string }>('/settings/github-token');
  }

  // Services
  async listServices(projectId: string): Promise<Service[]> {
    const result = await this.get<ApiResponse<Service[]>>(`/projects/${projectId}/services`);
    return result.data;
  }

  async getService(projectId: string, serviceName: string): Promise<Service> {
    const result = await this.get<ApiResponse<Service>>(`/projects/${projectId}/services/${serviceName}`);
    return result.data;
  }

  async getServiceById(serviceId: string): Promise<Service> {
    const result = await this.get<ApiResponse<Service>>(`/services/${serviceId}`);
    return result.data;
  }

  async createService(projectId: string, data: CreateServiceRequest): Promise<Service> {
    const result = await this.post<ApiResponse<Service>>(`/projects/${projectId}/services`, data);
    return result.data;
  }

  async updateService(projectId: string, serviceName: string, data: UpdateServiceRequest): Promise<Service> {
    const result = await this.put<ApiResponse<Service>>(`/projects/${projectId}/services/${serviceName}`, data);
    return result.data;
  }

  async deleteService(projectId: string, serviceName: string): Promise<void> {
    await this.delete(`/projects/${projectId}/services/${serviceName}`);
  }

  async deployService(projectId: string, serviceName: string, environment?: Record<string, string>): Promise<Deployment> {
    const result = await this.post<ApiResponse<Deployment>>(`/projects/${projectId}/services/${serviceName}/deploy`, { environment });
    return result.data;
  }

  async listServiceDeployments(projectId: string, serviceName: string): Promise<Deployment[]> {
    const result = await this.get<ApiResponse<Deployment[]>>(`/projects/${projectId}/services/${serviceName}/deployments`);
    return result.data;
  }

  // Domains
  async listProjectDomains(projectId: string): Promise<Domain[]> {
    const result = await this.get<ApiResponse<Domain[]>>(`/projects/${projectId}/domains`);
    return result.data;
  }

  async listServiceDomains(projectId: string, serviceName: string): Promise<Domain[]> {
    const result = await this.get<ApiResponse<Domain[]>>(`/projects/${projectId}/services/${serviceName}/domains`);
    return result.data;
  }

  async getDomain(domainName: string): Promise<Domain> {
    const result = await this.get<ApiResponse<Domain>>(`/domains/${domainName}`);
    return result.data;
  }

  async createDomain(projectId: string, serviceName: string, data: CreateDomainRequest): Promise<Domain> {
    const result = await this.post<ApiResponse<Domain>>(`/projects/${projectId}/services/${serviceName}/domains`, data);
    return result.data;
  }

  async updateDomain(domainName: string, data: UpdateDomainRequest): Promise<Domain> {
    const result = await this.put<ApiResponse<Domain>>(`/domains/${domainName}`, data);
    return result.data;
  }

  async deleteDomain(domainName: string): Promise<void> {
    await this.delete(`/domains/${domainName}`);
  }
}

// Types

// Legacy App type (maps to Project + main service)
export interface App {
  id: string;
  name: string;
  display_name?: string;
  description?: string;
  deployment_mode: 'git' | 'docker_image' | 'docker_compose';
  domain: string;
  git_repo?: string;
  git_branch?: string;
  docker_image?: string;
  compose_file?: string;
  environment: Record<string, string>;
  created_at: string;
  updated_at: string;
}

// New Project type
export interface Project {
  id: string;
  name: string;
  display_name?: string;
  description?: string;
  git_repo?: string;
  git_branch?: string;
  environment: Record<string, string>;
  created_at: string;
  updated_at: string;
}

// Service types
export type ServiceType = 'web' | 'worker' | 'cron' | 'database';
export type BuilderType = 'nixpacks' | 'railpacks' | 'dockerfile' | 'docker_image' | 'buildpacks';

export interface Service {
  id: string;
  project_id: string;
  name: string;
  type: ServiceType;
  builder: BuilderType;
  git_repo?: string;
  git_branch?: string;
  subdirectory?: string;
  docker_image?: string;
  database_type?: string;
  database_version?: string;
  // Database connection info
  database_host?: string;
  database_port?: number;
  database_user?: string;
  database_password?: string;
  database_name?: string;
  database_exposed?: boolean;
  port: number;
  command?: string;
  environment: Record<string, string>;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateServiceRequest {
  name: string;
  type?: ServiceType;
  builder?: BuilderType;
  git_repo?: string;
  git_branch?: string;
  subdirectory?: string;
  docker_image?: string;
  database_type?: string;
  database_version?: string;
  port?: number;
  command?: string;
  environment?: Record<string, string>;
}

export interface UpdateServiceRequest {
  builder?: BuilderType;
  git_repo?: string;
  git_branch?: string;
  subdirectory?: string;
  docker_image?: string;
  database_version?: string;
  port?: number;
  command?: string;
  environment?: Record<string, string>;
}

// Domain types
export interface Domain {
  id: string;
  project_id: string;
  service_id: string;
  domain: string;
  path_prefix: string;
  active_slot: 'blue' | 'green';
  ssl_enabled: boolean;
  created_at: string;
}

export interface CreateDomainRequest {
  domain: string;
  path_prefix?: string;
  ssl_enabled?: boolean;
}

export interface UpdateDomainRequest {
  path_prefix?: string;
  ssl_enabled?: boolean;
}

export interface Deployment {
  id: string;
  app_id: string;
  service_id?: string;
  version: string;
  slot: 'blue' | 'green';
  status: 'pending' | 'preparing' | 'deploying' | 'running' | 'stopped' | 'failed';
  error_message?: string;
  container_ids?: string[];
  created_at: string;
  finished_at?: string;
}

export interface CreateAppRequest {
  name: string;
  display_name?: string;
  description?: string;
  deployment_mode?: 'git' | 'docker_image' | 'docker_compose';
  domain?: string;
  git_repo?: string;
  git_branch?: string;
  docker_image?: string;
  compose_file?: string;
  environment?: Record<string, string>;
}

export interface DeployImageRequest {
  image: string;  // Can include tag like "nginx:latest"
  port: number;
  registry?: string;
  registry_auth?: {
    username: string;
    password: string;
  };
  pull_policy?: string;
  environment?: Record<string, string>;
}

export interface DeployGitRequest {
  branch?: string;
  environment?: Record<string, string>;
}

export interface GitHubTokenStatus {
  configured: boolean;
}

// Real-time status event from SSE stream
export interface StatusEvent {
  type: 'deployment_status' | 'service_status';
  deployment_id?: string;
  service_id?: string;
  project_id: string;
  status: string;
  error_message?: string;
  timestamp: string;
}

export const api = new ApiClient();
export default api;
