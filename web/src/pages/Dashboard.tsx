import { Component, createSignal, onMount, For, Show } from 'solid-js';
import api, { App } from '../api/client';
import CreateAppModal from '../components/CreateAppModal';

interface DashboardProps {
  onSelectApp: (name: string) => void;
}

const Dashboard: Component<DashboardProps> = (props) => {
  const [apps, setApps] = createSignal<App[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [showCreateModal, setShowCreateModal] = createSignal(false);

  const fetchApps = async () => {
    try {
      const data = await api.listApps();
      setApps(data);
    } catch (e) {
      console.error('Failed to fetch apps:', e);
    } finally {
      setLoading(false);
    }
  };

  onMount(fetchApps);

  const handleAppCreated = () => {
    setShowCreateModal(false);
    fetchApps();
  };

  const getModeIcon = (mode: string) => {
    switch (mode) {
      case 'git':
        return (
          <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
          </svg>
        );
      case 'docker_image':
        return (
          <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M13.983 11.078h2.119a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.119a.185.185 0 00-.185.185v1.888c0 .102.083.185.185.185m-2.954-5.43h2.118a.186.186 0 00.186-.186V3.574a.186.186 0 00-.186-.185h-2.118a.185.185 0 00-.185.185v1.888c0 .102.082.185.185.185m0 2.716h2.118a.187.187 0 00.186-.186V6.29a.186.186 0 00-.186-.185h-2.118a.185.185 0 00-.185.185v1.887c0 .102.082.186.185.186m-2.93 0h2.12a.186.186 0 00.184-.186V6.29a.185.185 0 00-.185-.185H8.1a.185.185 0 00-.185.185v1.887c0 .102.083.186.185.186m-2.964 0h2.119a.186.186 0 00.185-.186V6.29a.185.185 0 00-.185-.185H5.136a.186.186 0 00-.186.185v1.887c0 .102.084.186.186.186m5.893 2.715h2.118a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.118a.185.185 0 00-.185.185v1.888c0 .102.082.185.185.185m-2.93 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.185v1.888c0 .102.083.185.185.185m-2.964 0h2.119a.185.185 0 00.185-.185V9.006a.185.185 0 00-.185-.186H5.136a.186.186 0 00-.186.186v1.887c0 .102.084.185.186.185m-2.92 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.185v1.888c0 .102.082.185.185.185M23.763 9.89c-.065-.051-.672-.51-1.954-.51-.338.001-.676.03-1.01.087-.248-1.7-1.653-2.53-1.716-2.566l-.344-.199-.226.327c-.284.438-.49.922-.612 1.43-.23.97-.09 1.882.403 2.661-.595.332-1.55.413-1.744.42H.751a.751.751 0 00-.75.748 11.376 11.376 0 00.692 4.062c.545 1.428 1.355 2.48 2.41 3.124 1.18.723 3.1 1.137 5.275 1.137.983.003 1.963-.086 2.93-.266a12.248 12.248 0 003.823-1.389c.98-.567 1.86-1.288 2.61-2.136 1.252-1.418 1.998-2.997 2.553-4.4h.221c1.372 0 2.215-.549 2.68-1.009.309-.293.55-.65.707-1.046l.098-.288z"/>
          </svg>
        );
      case 'docker_compose':
        return (
          <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M3 3h8v8H3V3zm10 0h8v8h-8V3zM3 13h8v8H3v-8zm10 0h8v8h-8v-8z"/>
          </svg>
        );
      default:
        return null;
    }
  };

  const getStatusColor = (_app: App) => {
    return 'bg-green-100 text-green-800';
  };

  return (
    <div>
      <div class="flex items-center justify-between mb-8">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">Aplicaciones</h1>
          <p class="text-gray-600">Gestiona tus aplicaciones desplegadas</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          class="btn btn-primary flex items-center space-x-2"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <span>Nueva App</span>
        </button>
      </div>

      <Show when={!loading()} fallback={<LoadingGrid />}>
        <Show
          when={apps().length > 0}
          fallback={<EmptyState onCreateClick={() => setShowCreateModal(true)} />}
        >
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <For each={apps()}>
              {(app) => (
                <button
                  onClick={() => props.onSelectApp(app.name)}
                  class="card hover:shadow-md transition-shadow text-left"
                >
                  <div class="flex items-start justify-between mb-4">
                    <div class="flex items-center space-x-3">
                      <div class="p-2 bg-nebula-100 text-nebula-600 rounded-lg">
                        {getModeIcon(app.deployment_mode)}
                      </div>
                      <div>
                        <h3 class="font-semibold text-gray-900">{app.name}</h3>
                        <p class="text-sm text-gray-500">{app.deployment_mode}</p>
                      </div>
                    </div>
                    <span class={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(app)}`}>
                      Activo
                    </span>
                  </div>
                  {app.domain && (
                    <p class="text-sm text-nebula-600 truncate">
                      https://{app.domain}
                    </p>
                  )}
                  <p class="text-xs text-gray-400 mt-2">
                    Creado {new Date(app.created_at).toLocaleDateString()}
                  </p>
                </button>
              )}
            </For>
          </div>
        </Show>
      </Show>

      <Show when={showCreateModal()}>
        <CreateAppModal
          onClose={() => setShowCreateModal(false)}
          onCreated={handleAppCreated}
        />
      </Show>
    </div>
  );
};

const LoadingGrid: Component = () => (
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
    <For each={[1, 2, 3]}>
      {() => (
        <div class="card animate-pulse">
          <div class="flex items-center space-x-3 mb-4">
            <div class="w-10 h-10 bg-gray-200 rounded-lg"></div>
            <div class="flex-1">
              <div class="h-4 bg-gray-200 rounded w-24 mb-2"></div>
              <div class="h-3 bg-gray-200 rounded w-16"></div>
            </div>
          </div>
          <div class="h-3 bg-gray-200 rounded w-32"></div>
        </div>
      )}
    </For>
  </div>
);

const EmptyState: Component<{ onCreateClick: () => void }> = (props) => (
  <div class="text-center py-16">
    <svg class="w-16 h-16 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
    </svg>
    <h3 class="text-lg font-medium text-gray-900 mb-2">Sin aplicaciones</h3>
    <p class="text-gray-600 mb-6">Comienza creando tu primera aplicación</p>
    <button onClick={props.onCreateClick} class="btn btn-primary">
      Crear Aplicación
    </button>
  </div>
);

export default Dashboard;
