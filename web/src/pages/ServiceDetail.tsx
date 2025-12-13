import { Component, createSignal, onMount, For, Show, createMemo } from 'solid-js';
import api, { Service, Deployment, StatusEvent } from '../api/client';
import DeploymentLogsModal from '../components/DeploymentLogsModal';
import EditEnvVarsModal from '../components/EditEnvVarsModal';
import { useProjectStatusStream } from '../hooks/useStatusStream';

interface ServiceDetailProps {
  projectId: string;
  projectName: string;
  serviceName: string;
  onBack: () => void;
}

const ServiceDetail: Component<ServiceDetailProps> = (props) => {
  const [service, setService] = createSignal<Service | null>(null);
  const [deployments, setDeployments] = createSignal<Deployment[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [deploying, setDeploying] = createSignal(false);
  const [selectedDeployment, setSelectedDeployment] = createSignal<Deployment | null>(null);
  const [showPassword, setShowPassword] = createSignal(false);
  const [copiedField, setCopiedField] = createSignal<string | null>(null);
  const [showEnvModal, setShowEnvModal] = createSignal(false);

  const fetchData = async () => {
    try {
      const [svc, deps] = await Promise.all([
        api.getService(props.projectId, props.serviceName),
        api.listServiceDeployments(props.projectId, props.serviceName),
      ]);
      setService(svc);
      setDeployments(deps);
    } catch (e) {
      console.error('Failed to fetch service:', e);
    } finally {
      setLoading(false);
    }
  };

  // Real-time status updates via SSE
  useProjectStatusStream(
    () => props.projectId,
    (event: StatusEvent) => {
      // Update deployment status
      if (event.type === 'deployment_status' && event.deployment_id) {
        setDeployments(prev => prev.map(d =>
          d.id === event.deployment_id
            ? { ...d, status: event.status as Deployment['status'], error_message: event.error_message }
            : d
        ));
      }
      // Update service status
      if (event.type === 'service_status' && event.service_id === service()?.id) {
        setService(prev => prev ? { ...prev, status: event.status } : null);
      }
    }
  );

  onMount(fetchData);

  const handleDeploy = async () => {
    setDeploying(true);
    try {
      await api.deployService(props.projectId, props.serviceName);
      fetchData();
    } catch (e) {
      alert('Error al desplegar: ' + (e instanceof Error ? e.message : 'Error desconocido'));
    } finally {
      setDeploying(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const colors: Record<string, string> = {
      running: 'bg-green-100 text-green-700',
      stopped: 'bg-gray-100 text-gray-600',
      failed: 'bg-red-100 text-red-700',
      building: 'bg-yellow-100 text-yellow-700',
      pending: 'bg-blue-100 text-blue-700',
      preparing: 'bg-blue-100 text-blue-700',
      deploying: 'bg-yellow-100 text-yellow-700',
    };
    return colors[status] || 'bg-gray-100 text-gray-600';
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'web':
        return (
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
        );
      case 'worker':
        return (
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        );
      case 'cron':
        return (
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      case 'database':
        return (
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
          </svg>
        );
      default:
        return null;
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const copyToClipboard = async (text: string, field: string) => {
    await navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const getConnectionString = () => {
    const svc = service();
    if (!svc || svc.type !== 'database') return '';

    const { database_type, database_host, database_port, database_user, database_password, database_name } = svc;

    switch (database_type) {
      case 'postgres':
        return `postgresql://${database_user}:${database_password}@${database_host}:${database_port}/${database_name}`;
      case 'mysql':
        return `mysql://${database_user}:${database_password}@${database_host}:${database_port}/${database_name}`;
      case 'mongodb':
        return `mongodb://${database_user}:${database_password}@${database_host}:${database_port}/${database_name}?authSource=admin`;
      case 'redis':
        return `redis://${database_host}:${database_port}`;
      default:
        return '';
    }
  };

  const handleSaveEnvVars = async (env: Record<string, string>) => {
    await api.updateService(props.projectId, props.serviceName, { environment: env });
    // Refresh service data
    const updatedService = await api.getService(props.projectId, props.serviceName);
    setService(updatedService);
  };

  return (
    <Show when={!loading()} fallback={<LoadingSkeleton />}>
      <Show when={service()}>
        <div>
          {/* Header */}
          <div class="flex items-center justify-between mb-8">
            <div class="flex items-center space-x-4">
              <button
                onClick={props.onBack}
                class="p-2 hover:bg-gray-100 rounded-lg"
              >
                <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
                </svg>
              </button>
              <div class="p-3 bg-nebula-100 rounded-lg text-nebula-600">
                {getTypeIcon(service()!.type)}
              </div>
              <div>
                <h1 class="text-2xl font-bold text-gray-900">{service()!.name}</h1>
                <p class="text-gray-500 text-sm">{props.projectName} / {service()!.type}</p>
              </div>
            </div>
            <div class="flex items-center space-x-3">
              <span class={`px-3 py-1 text-sm font-medium rounded-full ${getStatusBadge(service()!.status)}`}>
                {service()!.status}
              </span>
              <button
                onClick={handleDeploy}
                disabled={deploying()}
                class="btn btn-primary"
              >
                {deploying() ? 'Desplegando...' : 'Desplegar'}
              </button>
            </div>
          </div>

          {/* Service Info */}
          <div class="card mb-6">
            <h3 class="font-semibold text-gray-900 mb-4">Configuracion del Servicio</h3>
            <dl class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <dt class="text-gray-500">Tipo</dt>
                <dd class="text-gray-700 capitalize">{service()!.type}</dd>
              </div>
              {service()!.type === 'database' ? (
                <>
                  <div>
                    <dt class="text-gray-500">Base de Datos</dt>
                    <dd class="text-gray-700">{service()!.database_type}</dd>
                  </div>
                  <div>
                    <dt class="text-gray-500">Version</dt>
                    <dd class="text-gray-700">{service()!.database_version || 'latest'}</dd>
                  </div>
                </>
              ) : (
                <>
                  <div>
                    <dt class="text-gray-500">Builder</dt>
                    <dd class="text-gray-700">{service()!.builder}</dd>
                  </div>
                  <div>
                    <dt class="text-gray-500">Puerto</dt>
                    <dd class="text-gray-700">{service()!.port || 'N/A'}</dd>
                  </div>
                </>
              )}
              <div>
                <dt class="text-gray-500">Creado</dt>
                <dd class="text-gray-700">{formatDate(service()!.created_at)}</dd>
              </div>
              {service()!.docker_image && (
                <div class="col-span-2">
                  <dt class="text-gray-500">Imagen Docker</dt>
                  <dd class="font-mono text-gray-700 truncate">{service()!.docker_image}</dd>
                </div>
              )}
              {service()!.git_repo && (
                <div class="col-span-2">
                  <dt class="text-gray-500">Repositorio Git</dt>
                  <dd class="font-mono text-gray-700 truncate">{service()!.git_repo}</dd>
                </div>
              )}
              {service()!.git_branch && (
                <div>
                  <dt class="text-gray-500">Rama</dt>
                  <dd class="text-gray-700">{service()!.git_branch}</dd>
                </div>
              )}
            </dl>
          </div>

          {/* Environment Variables Section */}
          <div class="card mb-6">
            <div class="flex items-center justify-between mb-4">
              <h3 class="font-semibold text-gray-900">Variables de Entorno</h3>
              <button
                type="button"
                onClick={() => setShowEnvModal(true)}
                class="text-sm text-nebula-600 hover:text-nebula-700 font-medium"
              >
                Editar
              </button>
            </div>
            <Show
              when={Object.keys(service()!.environment || {}).length > 0}
              fallback={
                <p class="text-gray-400 text-sm">No hay variables de entorno configuradas</p>
              }
            >
              <div class="space-y-2">
                <For each={Object.entries(service()!.environment || {})}>
                  {([key, value]) => (
                    <div class="flex items-center justify-between py-2 px-3 bg-gray-50 rounded-lg">
                      <span class="font-mono text-sm text-gray-700">{key}</span>
                      <span class="font-mono text-sm text-gray-400">••••••••</span>
                    </div>
                  )}
                </For>
              </div>
              <p class="text-xs text-gray-500 mt-3">
                Los valores se ocultan por seguridad. Haz clic en "Editar" para ver o modificar.
              </p>
            </Show>
          </div>

          {/* Database Connection Info */}
          <Show when={service()!.type === 'database' && service()!.database_host}>
            <div class="card mb-6">
              <h3 class="font-semibold text-gray-900 mb-4">Informacion de Conexion</h3>
              <div class="space-y-4">
                {/* Connection String */}
                <div>
                  <label class="text-sm text-gray-500 block mb-1">Connection String</label>
                  <div class="flex items-center bg-gray-50 rounded-lg p-3 font-mono text-sm">
                    <span class="flex-1 truncate">
                      {showPassword() ? getConnectionString() : getConnectionString().replace(service()!.database_password || '', '••••••••')}
                    </span>
                    <button
                      onClick={() => copyToClipboard(getConnectionString(), 'connection')}
                      class="ml-2 p-1 hover:bg-gray-200 rounded"
                      title="Copiar"
                    >
                      <Show when={copiedField() === 'connection'} fallback={
                        <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                      }>
                        <svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                        </svg>
                      </Show>
                    </button>
                  </div>
                </div>

                {/* Individual Fields */}
                <div class="grid grid-cols-2 md:grid-cols-3 gap-4">
                  <div>
                    <label class="text-sm text-gray-500 block mb-1">Host</label>
                    <div class="flex items-center">
                      <span class="font-mono text-sm text-gray-700">{service()!.database_host}</span>
                      <button
                        onClick={() => copyToClipboard(service()!.database_host || '', 'host')}
                        class="ml-2 p-1 hover:bg-gray-100 rounded"
                      >
                        <Show when={copiedField() === 'host'} fallback={
                          <svg class="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                          </svg>
                        }>
                          <svg class="w-3 h-3 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                          </svg>
                        </Show>
                      </button>
                    </div>
                  </div>

                  <div>
                    <label class="text-sm text-gray-500 block mb-1">Puerto</label>
                    <span class="font-mono text-sm text-gray-700">{service()!.database_port}</span>
                  </div>

                  <Show when={service()!.database_user}>
                    <div>
                      <label class="text-sm text-gray-500 block mb-1">Usuario</label>
                      <div class="flex items-center">
                        <span class="font-mono text-sm text-gray-700">{service()!.database_user}</span>
                        <button
                          onClick={() => copyToClipboard(service()!.database_user || '', 'user')}
                          class="ml-2 p-1 hover:bg-gray-100 rounded"
                        >
                          <Show when={copiedField() === 'user'} fallback={
                            <svg class="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                            </svg>
                          }>
                            <svg class="w-3 h-3 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                            </svg>
                          </Show>
                        </button>
                      </div>
                    </div>
                  </Show>

                  <Show when={service()!.database_password}>
                    <div>
                      <label class="text-sm text-gray-500 block mb-1">Contrasena</label>
                      <div class="flex items-center">
                        <span class="font-mono text-sm text-gray-700">
                          {showPassword() ? service()!.database_password : '••••••••'}
                        </span>
                        <button
                          onClick={() => setShowPassword(!showPassword())}
                          class="ml-2 p-1 hover:bg-gray-100 rounded"
                          title={showPassword() ? 'Ocultar' : 'Mostrar'}
                        >
                          <Show when={showPassword()} fallback={
                            <svg class="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                            </svg>
                          }>
                            <svg class="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                            </svg>
                          </Show>
                        </button>
                        <button
                          onClick={() => copyToClipboard(service()!.database_password || '', 'password')}
                          class="ml-1 p-1 hover:bg-gray-100 rounded"
                        >
                          <Show when={copiedField() === 'password'} fallback={
                            <svg class="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                            </svg>
                          }>
                            <svg class="w-3 h-3 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                            </svg>
                          </Show>
                        </button>
                      </div>
                    </div>
                  </Show>

                  <Show when={service()!.database_name}>
                    <div>
                      <label class="text-sm text-gray-500 block mb-1">Base de Datos</label>
                      <div class="flex items-center">
                        <span class="font-mono text-sm text-gray-700">{service()!.database_name}</span>
                        <button
                          onClick={() => copyToClipboard(service()!.database_name || '', 'dbname')}
                          class="ml-2 p-1 hover:bg-gray-100 rounded"
                        >
                          <Show when={copiedField() === 'dbname'} fallback={
                            <svg class="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                            </svg>
                          }>
                            <svg class="w-3 h-3 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                            </svg>
                          </Show>
                        </button>
                      </div>
                    </div>
                  </Show>
                </div>

                {/* Note about internal network */}
                <div class="text-xs text-gray-500 bg-yellow-50 p-3 rounded-lg">
                  <strong>Nota:</strong> Esta base de datos solo es accesible desde otros servicios en la red interna de Nebula.
                  El host <code class="bg-yellow-100 px-1 rounded">{service()!.database_host}</code> solo resuelve dentro de la red Docker.
                </div>
              </div>
            </div>
          </Show>

          {/* Deployments Section */}
          <div class="mb-4">
            <h2 class="text-lg font-semibold text-gray-900">Historial de Despliegues</h2>
          </div>

          <Show
            when={deployments().length > 0}
            fallback={
              <div class="card text-center py-12">
                <svg class="w-12 h-12 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
                <h3 class="text-gray-600 font-medium mb-2">No hay despliegues</h3>
                <p class="text-gray-500 text-sm mb-4">
                  Despliega este servicio para ver el historial
                </p>
                <button onClick={handleDeploy} class="btn btn-primary">
                  Desplegar Ahora
                </button>
              </div>
            }
          >
            <div class="card overflow-hidden">
              <table class="min-w-full divide-y divide-gray-200">
                <thead class="bg-gray-50">
                  <tr>
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Version</th>
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Slot</th>
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Estado</th>
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Fecha</th>
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Acciones</th>
                  </tr>
                </thead>
                <tbody class="bg-white divide-y divide-gray-200">
                  <For each={deployments()}>
                    {(deployment) => (
                      <>
                        <tr class="hover:bg-gray-50">
                          <td class="px-4 py-3 text-sm font-mono text-gray-900">{deployment.version}</td>
                          <td class="px-4 py-3 text-sm">
                            <span class={`px-2 py-1 text-xs rounded ${deployment.slot === 'blue' ? 'bg-blue-100 text-blue-700' : 'bg-green-100 text-green-700'}`}>
                              {deployment.slot}
                            </span>
                          </td>
                          <td class="px-4 py-3 text-sm">
                            <span class={`px-2 py-1 text-xs rounded-full ${getStatusBadge(deployment.status)}`}>
                              {deployment.status}
                            </span>
                          </td>
                          <td class="px-4 py-3 text-sm text-gray-500">{formatDate(deployment.created_at)}</td>
                          <td class="px-4 py-3 text-sm">
                            <button
                              onClick={() => setSelectedDeployment(deployment)}
                              class="text-nebula-600 hover:text-nebula-800 font-medium"
                            >
                              Ver Logs
                            </button>
                          </td>
                        </tr>
                        <Show when={deployment.status === 'failed' && deployment.error_message}>
                          <tr class="bg-red-50">
                            <td colspan="5" class="px-4 py-2 text-sm text-red-700">
                              <span class="font-medium">Error:</span> {deployment.error_message}
                            </td>
                          </tr>
                        </Show>
                      </>
                    )}
                  </For>
                </tbody>
              </table>
            </div>
          </Show>

          {/* Deployment Logs Modal */}
          <Show when={selectedDeployment()}>
            <DeploymentLogsModal
              appName={props.projectName}
              deploymentId={selectedDeployment()!.id}
              deploymentVersion={selectedDeployment()!.version}
              onClose={() => setSelectedDeployment(null)}
            />
          </Show>

          {/* Edit Environment Variables Modal */}
          <Show when={showEnvModal()}>
            <EditEnvVarsModal
              environment={service()!.environment || {}}
              onSave={handleSaveEnvVars}
              onClose={() => setShowEnvModal(false)}
            />
          </Show>
        </div>
      </Show>
    </Show>
  );
};

const LoadingSkeleton: Component = () => (
  <div class="animate-pulse">
    <div class="flex items-center space-x-4 mb-8">
      <div class="h-10 w-10 bg-gray-200 rounded-lg"></div>
      <div>
        <div class="h-6 bg-gray-200 rounded w-48 mb-2"></div>
        <div class="h-4 bg-gray-200 rounded w-32"></div>
      </div>
    </div>
    <div class="card mb-6">
      <div class="h-4 bg-gray-200 rounded w-32 mb-4"></div>
      <div class="grid grid-cols-4 gap-4">
        <div class="h-12 bg-gray-200 rounded"></div>
        <div class="h-12 bg-gray-200 rounded"></div>
        <div class="h-12 bg-gray-200 rounded"></div>
        <div class="h-12 bg-gray-200 rounded"></div>
      </div>
    </div>
    <div class="h-6 bg-gray-200 rounded w-48 mb-4"></div>
    <div class="card">
      <div class="h-32 bg-gray-200 rounded"></div>
    </div>
  </div>
);

export default ServiceDetail;
