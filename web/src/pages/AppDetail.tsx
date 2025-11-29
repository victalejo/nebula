import { Component, createSignal, onMount, For, Show } from 'solid-js';
import api, { App, Deployment } from '../api/client';
import DeployModal from '../components/DeployModal';
import LogViewer from '../components/LogViewer';

interface AppDetailProps {
  appName: string;
  onBack: () => void;
}

const AppDetail: Component<AppDetailProps> = (props) => {
  const [app, setApp] = createSignal<App | null>(null);
  const [deployments, setDeployments] = createSignal<Deployment[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [activeTab, setActiveTab] = createSignal<'overview' | 'deployments' | 'logs'>('overview');
  const [showDeployModal, setShowDeployModal] = createSignal(false);
  const [deleting, setDeleting] = createSignal(false);

  const fetchData = async () => {
    try {
      const [appData, deploymentsData] = await Promise.all([
        api.getApp(props.appName),
        api.listDeployments(props.appName),
      ]);
      setApp(appData);
      setDeployments(deploymentsData);
    } catch (e) {
      console.error('Failed to fetch app:', e);
    } finally {
      setLoading(false);
    }
  };

  onMount(fetchData);

  const handleDeploy = () => {
    setShowDeployModal(false);
    fetchData();
  };

  const handleDelete = async () => {
    if (!confirm(`¿Estás seguro de que deseas eliminar ${props.appName}? Esta acción no se puede deshacer.`)) {
      return;
    }

    setDeleting(true);
    try {
      await api.deleteApp(props.appName);
      props.onBack();
    } catch (e) {
      alert('Error al eliminar la aplicación');
    } finally {
      setDeleting(false);
    }
  };

  const getCurrentDeployment = () => {
    return deployments().find((d) => d.status === 'running');
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <span class="text-green-500">●</span>;
      case 'stopped':
        return <span class="text-gray-400">○</span>;
      case 'failed':
        return <span class="text-red-500">✗</span>;
      default:
        return <span class="text-yellow-500">◐</span>;
    }
  };

  return (
    <Show when={!loading()} fallback={<LoadingSkeleton />}>
      <Show when={app()}>
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
              <div>
                <h1 class="text-2xl font-bold text-gray-900">{app()!.name}</h1>
                {app()!.domain && (
                  <a
                    href={`https://${app()!.domain}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-nebula-600 hover:underline text-sm"
                  >
                    https://{app()!.domain}
                  </a>
                )}
              </div>
            </div>
            <div class="flex items-center space-x-3">
              <button
                onClick={() => setShowDeployModal(true)}
                class="btn btn-primary"
              >
                Desplegar
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting()}
                class="btn btn-danger"
              >
                {deleting() ? 'Eliminando...' : 'Eliminar'}
              </button>
            </div>
          </div>

          {/* Tabs */}
          <div class="border-b border-gray-200 mb-6">
            <nav class="flex space-x-8">
              <TabButton
                active={activeTab() === 'overview'}
                onClick={() => setActiveTab('overview')}
              >
                General
              </TabButton>
              <TabButton
                active={activeTab() === 'deployments'}
                onClick={() => setActiveTab('deployments')}
              >
                Despliegues
              </TabButton>
              <TabButton
                active={activeTab() === 'logs'}
                onClick={() => setActiveTab('logs')}
              >
                Registros
              </TabButton>
            </nav>
          </div>

          {/* Tab Content */}
          <Show when={activeTab() === 'overview'}>
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* App Info */}
              <div class="card">
                <h3 class="font-semibold text-gray-900 mb-4">Info de Aplicación</h3>
                <dl class="space-y-3">
                  <InfoRow label="Modo" value={app()!.deployment_mode} />
                  <InfoRow label="Dominio" value={app()!.domain || 'No configurado'} />
                  {app()!.docker_image && (
                    <InfoRow label="Imagen" value={app()!.docker_image!} />
                  )}
                  {app()!.git_repo && (
                    <>
                      <InfoRow label="Repositorio" value={app()!.git_repo!} />
                      <InfoRow label="Rama" value={app()!.git_branch || 'main'} />
                    </>
                  )}
                  <InfoRow
                    label="Creado"
                    value={new Date(app()!.created_at).toLocaleString()}
                  />
                </dl>
              </div>

              {/* Current Deployment */}
              <div class="card">
                <h3 class="font-semibold text-gray-900 mb-4">Despliegue Actual</h3>
                <Show
                  when={getCurrentDeployment()}
                  fallback={<p class="text-gray-500">Sin despliegue activo</p>}
                >
                  {(deployment) => (
                    <dl class="space-y-3">
                      <InfoRow label="Versión" value={deployment().version} />
                      <InfoRow label="Slot" value={deployment().slot} />
                      <InfoRow
                        label="Estado"
                        value={
                          <span class="flex items-center space-x-2">
                            {getStatusIcon(deployment().status)}
                            <span>{deployment().status}</span>
                          </span>
                        }
                      />
                      <InfoRow
                        label="Desplegado"
                        value={new Date(deployment().created_at).toLocaleString()}
                      />
                    </dl>
                  )}
                </Show>
              </div>

              {/* Environment Variables */}
              <div class="card lg:col-span-2">
                <h3 class="font-semibold text-gray-900 mb-4">Variables de Entorno</h3>
                <Show
                  when={Object.keys(app()!.env_vars || {}).length > 0}
                  fallback={<p class="text-gray-500">Sin variables de entorno</p>}
                >
                  <div class="bg-gray-50 rounded-lg p-4 font-mono text-sm">
                    <For each={Object.entries(app()!.env_vars || {})}>
                      {([key, value]) => (
                        <div class="flex">
                          <span class="text-nebula-600">{key}</span>
                          <span class="text-gray-400">=</span>
                          <span class="text-gray-700">{value}</span>
                        </div>
                      )}
                    </For>
                  </div>
                </Show>
              </div>
            </div>
          </Show>

          <Show when={activeTab() === 'deployments'}>
            <div class="card">
              <Show
                when={deployments().length > 0}
                fallback={<p class="text-gray-500 text-center py-8">Sin despliegues</p>}
              >
                <table class="w-full">
                  <thead>
                    <tr class="text-left text-sm text-gray-500 border-b">
                      <th class="pb-3">Versión</th>
                      <th class="pb-3">Slot</th>
                      <th class="pb-3">Estado</th>
                      <th class="pb-3">Creado</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y">
                    <For each={deployments()}>
                      {(deployment) => (
                        <tr>
                          <td class="py-3 font-mono text-sm">{deployment.version}</td>
                          <td class="py-3">
                            <span class={`px-2 py-1 text-xs rounded ${
                              deployment.slot === 'blue'
                                ? 'bg-blue-100 text-blue-700'
                                : 'bg-green-100 text-green-700'
                            }`}>
                              {deployment.slot}
                            </span>
                          </td>
                          <td class="py-3">
                            <span class="flex items-center space-x-2">
                              {getStatusIcon(deployment.status)}
                              <span>{deployment.status}</span>
                            </span>
                          </td>
                          <td class="py-3 text-sm text-gray-500">
                            {new Date(deployment.created_at).toLocaleString()}
                          </td>
                        </tr>
                      )}
                    </For>
                  </tbody>
                </table>
              </Show>
            </div>
          </Show>

          <Show when={activeTab() === 'logs'}>
            <LogViewer appName={props.appName} />
          </Show>

          <Show when={showDeployModal()}>
            <DeployModal
              app={app()!}
              onClose={() => setShowDeployModal(false)}
              onDeployed={handleDeploy}
            />
          </Show>
        </div>
      </Show>
    </Show>
  );
};

const TabButton: Component<{
  active: boolean;
  onClick: () => void;
  children: any;
}> = (props) => (
  <button
    onClick={props.onClick}
    class={`pb-4 px-1 border-b-2 font-medium text-sm transition-colors ${
      props.active
        ? 'border-nebula-500 text-nebula-600'
        : 'border-transparent text-gray-500 hover:text-gray-700'
    }`}
  >
    {props.children}
  </button>
);

const InfoRow: Component<{ label: string; value: any }> = (props) => (
  <div class="flex justify-between">
    <dt class="text-gray-500">{props.label}</dt>
    <dd class="text-gray-900 font-medium">{props.value}</dd>
  </div>
);

const LoadingSkeleton: Component = () => (
  <div class="animate-pulse">
    <div class="h-8 bg-gray-200 rounded w-48 mb-8"></div>
    <div class="grid grid-cols-2 gap-6">
      <div class="card">
        <div class="h-4 bg-gray-200 rounded w-32 mb-4"></div>
        <div class="space-y-3">
          <div class="h-4 bg-gray-200 rounded"></div>
          <div class="h-4 bg-gray-200 rounded"></div>
          <div class="h-4 bg-gray-200 rounded"></div>
        </div>
      </div>
      <div class="card">
        <div class="h-4 bg-gray-200 rounded w-32 mb-4"></div>
        <div class="space-y-3">
          <div class="h-4 bg-gray-200 rounded"></div>
          <div class="h-4 bg-gray-200 rounded"></div>
        </div>
      </div>
    </div>
  </div>
);

export default AppDetail;
