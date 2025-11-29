import { Component, createSignal, onMount, Show } from 'solid-js';
import { authStore } from './stores/auth';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import AppDetail from './pages/AppDetail';
import Settings from './pages/Settings';
import UpdateBanner from './components/UpdateBanner';

type Route = 'dashboard' | 'app-detail' | 'settings';

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

  const navigateToSettings = () => {
    setSelectedApp(null);
    setRoute('settings');
  };

  return (
    <Show when={initialized()} fallback={<LoadingScreen />}>
      <Show
        when={authStore.isAuthenticated()}
        fallback={<Login />}
      >
        <div class="min-h-screen bg-gray-50">
          <UpdateBanner />
          <Header onNavigate={navigateToDashboard} onSettings={navigateToSettings} />
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
            <Show when={route() === 'settings'}>
              <Settings onBack={navigateToDashboard} />
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

const Header: Component<{ onNavigate: () => void; onSettings: () => void }> = (props) => {
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
          <button
            onClick={props.onSettings}
            class="text-gray-500 hover:text-gray-700"
            title="Configuración"
          >
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>
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
