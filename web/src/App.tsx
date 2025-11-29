import { Component, createSignal, onMount, Show } from 'solid-js';
import { authStore } from './stores/auth';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import AppDetail from './pages/AppDetail';

type Route = 'dashboard' | 'app-detail';

const App: Component = () => {
  const [route, setRoute] = createSignal<Route>('dashboard');
  const [selectedApp, setSelectedApp] = createSignal<string | null>(null);
  const [initialized, setInitialized] = createSignal(false);

  onMount(async () => {
    await authStore.checkAuth();
    setInitialized(true);
  });

  const navigateToApp = (appName: string) => {
    setSelectedApp(appName);
    setRoute('app-detail');
  };

  const navigateToDashboard = () => {
    setSelectedApp(null);
    setRoute('dashboard');
  };

  return (
    <Show when={initialized()} fallback={<LoadingScreen />}>
      <Show
        when={authStore.isAuthenticated()}
        fallback={<Login />}
      >
        <div class="min-h-screen bg-gray-50">
          <Header onNavigate={navigateToDashboard} />
          <main class="max-w-7xl mx-auto px-4 py-8">
            <Show when={route() === 'dashboard'}>
              <Dashboard onSelectApp={navigateToApp} />
            </Show>
            <Show when={route() === 'app-detail' && selectedApp()}>
              <AppDetail
                appName={selectedApp()!}
                onBack={navigateToDashboard}
              />
            </Show>
          </main>
        </div>
      </Show>
    </Show>
  );
};

const LoadingScreen: Component = () => (
  <div class="min-h-screen bg-gray-50 flex items-center justify-center">
    <div class="text-center">
      <div class="w-12 h-12 border-4 border-nebula-500 border-t-transparent rounded-full animate-spin mx-auto"></div>
      <p class="mt-4 text-gray-600">Cargando...</p>
    </div>
  </div>
);

const Header: Component<{ onNavigate: () => void }> = (props) => {
  return (
    <header class="bg-white shadow-sm border-b border-gray-100">
      <div class="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
        <button
          onClick={props.onNavigate}
          class="flex items-center space-x-2 hover:opacity-80"
        >
          <svg class="w-8 h-8 text-nebula-500" viewBox="0 0 24 24" fill="currentColor">
            <circle cx="12" cy="12" r="10" opacity="0.2" />
            <circle cx="12" cy="12" r="6" opacity="0.4" />
            <circle cx="12" cy="12" r="3" />
          </svg>
          <span class="text-xl font-bold text-gray-900">Nebula</span>
        </button>
        <div class="flex items-center space-x-4">
          <span class="text-sm text-gray-600">
            {authStore.user()?.username}
          </span>
          <button
            onClick={() => authStore.logout()}
            class="text-sm text-gray-500 hover:text-gray-700"
          >
            Cerrar sesión
          </button>
        </div>
      </div>
    </header>
  );
};

export default App;
