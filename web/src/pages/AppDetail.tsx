import { Component, createSignal, onMount, For, Show } from 'solid-js';
import api, { App, Service } from '../api/client';
import CreateServiceModal from '../components/CreateServiceModal';

interface AppDetailProps {
  appName: string;
  onBack: () => void;
}

const AppDetail: Component<AppDetailProps> = (props) => {
  const [app, setApp] = createSignal<App | null>(null);
  const [services, setServices] = createSignal<Service[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [showCreateService, setShowCreateService] = createSignal(false);
  const [deleting, setDeleting] = createSignal(false);

  const fetchData = async () => {
    try {
      const appData = await api.getApp(props.appName);
      setApp(appData);

      // Fetch services for this app
      const servicesData = await api.listServices(appData.id);
      setServices(servicesData);
    } catch (e) {
      console.error('Failed to fetch app:', e);
    } finally {
      setLoading(false);
    }
  };

  onMount(fetchData);

  const handleServiceCreated = () => {
    setShowCreateService(false);
    fetchData();
  };

  const handleDeleteApp = async () => {
    if (!confirm(`¿Eliminar "${props.appName}"? Esta accion no se puede deshacer.`)) {
      return;
    }

    setDeleting(true);
    try {
      await api.deleteApp(props.appName);
      props.onBack();
    } catch (e) {
      alert('Error al eliminar la aplicacion');
    } finally {
      setDeleting(false);
    }
  };

  const handleDeleteService = async (serviceName: string) => {
    if (!confirm(`¿Eliminar servicio "${serviceName}"?`)) {
      return;
    }

    try {
      await api.deleteService(app()!.id, serviceName);
      fetchData();
    } catch (e) {
      alert('Error al eliminar el servicio');
    }
  };

  const getStatusBadge = (status: string) => {
    const colors: Record<string, string> = {
      running: 'bg-green-100 text-green-700',
      stopped: 'bg-gray-100 text-gray-600',
      failed: 'bg-red-100 text-red-700',
      building: 'bg-yellow-100 text-yellow-700',
    };
    return colors[status] || 'bg-gray-100 text-gray-600';
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'web':
        return (
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
        );
      case 'worker':
        return (
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        );
      case 'cron':
        return (
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      case 'database':
        return (
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
          </svg>
        );
      default:
        return null;
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
                <h1 class="text-2xl font-bold text-gray-900">
                  {app()!.display_name || app()!.name}
                </h1>
                <p class="text-gray-500 text-sm">{app()!.name}</p>
              </div>
            </div>
            <div class="flex items-center space-x-3">
              <button
                onClick={() => setShowCreateService(true)}
                class="btn btn-primary"
              >
                + Nuevo Servicio
              </button>
              <button
                onClick={handleDeleteApp}
                disabled={deleting()}
                class="btn btn-danger"
              >
                {deleting() ? 'Eliminando...' : 'Eliminar App'}
              </button>
            </div>
          </div>

          {/* App Info Card */}
          <div class="card mb-6">
            <div class="flex items-start justify-between">
              <div>
                <h3 class="font-semibold text-gray-900 mb-2">Informacion de la Aplicacion</h3>
                <dl class="grid grid-cols-2 gap-x-8 gap-y-2 text-sm">
                  <div>
                    <dt class="text-gray-500">ID</dt>
                    <dd class="font-mono text-gray-700">{app()!.id}</dd>
                  </div>
                  <div>
                    <dt class="text-gray-500">Creado</dt>
                    <dd class="text-gray-700">{new Date(app()!.created_at).toLocaleDateString()}</dd>
                  </div>
                  {app()!.description && (
                    <div class="col-span-2">
                      <dt class="text-gray-500">Descripcion</dt>
                      <dd class="text-gray-700">{app()!.description}</dd>
                    </div>
                  )}
                </dl>
              </div>
              <div class="text-right">
                <span class="text-2xl font-bold text-nebula-600">{services().length}</span>
                <p class="text-sm text-gray-500">servicios</p>
              </div>
            </div>
          </div>

          {/* Services Section */}
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">Servicios</h2>
          </div>

          <Show
            when={services().length > 0}
            fallback={
              <div class="card text-center py-12">
                <svg class="w-12 h-12 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                </svg>
                <h3 class="text-gray-600 font-medium mb-2">No hay servicios</h3>
                <p class="text-gray-500 text-sm mb-4">
                  Crea tu primer servicio para empezar a desplegar
                </p>
                <button
                  onClick={() => setShowCreateService(true)}
                  class="btn btn-primary"
                >
                  Crear Servicio
                </button>
              </div>
            }
          >
            <div class="grid gap-4">
              <For each={services()}>
                {(service) => (
                  <div class="card hover:shadow-md transition-shadow">
                    <div class="flex items-center justify-between">
                      <div class="flex items-center space-x-4">
                        <div class="p-2 bg-gray-100 rounded-lg text-gray-600">
                          {getTypeIcon(service.type)}
                        </div>
                        <div>
                          <h3 class="font-semibold text-gray-900">{service.name}</h3>
                          <div class="flex items-center space-x-3 text-sm text-gray-500">
                            <span class="capitalize">{service.type}</span>
                            <span>•</span>
                            <span>{service.builder || 'N/A'}</span>
                            {service.port && (
                              <>
                                <span>•</span>
                                <span>Puerto {service.port}</span>
                              </>
                            )}
                          </div>
                        </div>
                      </div>
                      <div class="flex items-center space-x-4">
                        <span class={`px-3 py-1 text-xs font-medium rounded-full ${getStatusBadge(service.status)}`}>
                          {service.status}
                        </span>
                        <button
                          onClick={() => handleDeleteService(service.name)}
                          class="text-red-500 hover:text-red-700 p-1"
                          title="Eliminar servicio"
                        >
                          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                    </div>

                    {/* Service details */}
                    <div class="mt-4 pt-4 border-t border-gray-100">
                      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                        {service.docker_image && (
                          <div>
                            <dt class="text-gray-500">Imagen</dt>
                            <dd class="font-mono text-gray-700 truncate">{service.docker_image}</dd>
                          </div>
                        )}
                        {service.git_repo && (
                          <div class="col-span-2">
                            <dt class="text-gray-500">Repositorio</dt>
                            <dd class="font-mono text-gray-700 truncate">{service.git_repo}</dd>
                          </div>
                        )}
                        {service.git_branch && (
                          <div>
                            <dt class="text-gray-500">Rama</dt>
                            <dd class="text-gray-700">{service.git_branch}</dd>
                          </div>
                        )}
                        {service.database_type && (
                          <div>
                            <dt class="text-gray-500">Base de Datos</dt>
                            <dd class="text-gray-700">{service.database_type} {service.database_version}</dd>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                )}
              </For>
            </div>
          </Show>

          {/* Create Service Modal */}
          <Show when={showCreateService()}>
            <CreateServiceModal
              projectId={app()!.id}
              onClose={() => setShowCreateService(false)}
              onCreated={handleServiceCreated}
            />
          </Show>
        </div>
      </Show>
    </Show>
  );
};

const LoadingSkeleton: Component = () => (
  <div class="animate-pulse">
    <div class="h-8 bg-gray-200 rounded w-48 mb-8"></div>
    <div class="card mb-6">
      <div class="h-4 bg-gray-200 rounded w-32 mb-4"></div>
      <div class="space-y-3">
        <div class="h-4 bg-gray-200 rounded w-64"></div>
        <div class="h-4 bg-gray-200 rounded w-48"></div>
      </div>
    </div>
    <div class="h-6 bg-gray-200 rounded w-24 mb-4"></div>
    <div class="card">
      <div class="h-16 bg-gray-200 rounded"></div>
    </div>
  </div>
);

export default AppDetail;
