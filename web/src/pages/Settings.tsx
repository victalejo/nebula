import { Component, createSignal, onMount } from 'solid-js';
import api from '../api/client';
import { updateStore } from '../stores/update';

interface SettingsProps {
  onBack: () => void;
}

const Settings: Component<SettingsProps> = (props) => {
  const [tokenConfigured, setTokenConfigured] = createSignal(false);
  const [newToken, setNewToken] = createSignal('');
  const [loading, setLoading] = createSignal(true);
  const [saving, setSaving] = createSignal(false);
  const [message, setMessage] = createSignal<{ type: 'success' | 'error'; text: string } | null>(null);

  // Update configuration state
  const [updateMode, setUpdateMode] = createSignal<'auto' | 'notify' | 'disabled'>('notify');
  const [checkInterval, setCheckInterval] = createSignal(1440);
  const [savingUpdate, setSavingUpdate] = createSignal(false);

  onMount(async () => {
    await Promise.all([loadTokenStatus(), loadUpdateConfig()]);
  });

  const loadTokenStatus = async () => {
    try {
      setLoading(true);
      const status = await api.getGitHubTokenStatus();
      setTokenConfigured(status.configured);
    } catch (err) {
      setMessage({ type: 'error', text: 'Error al cargar configuración' });
    } finally {
      setLoading(false);
    }
  };

  const loadUpdateConfig = async () => {
    try {
      await updateStore.fetchSystemInfo();
      const info = updateStore.systemInfo();
      if (info) {
        setUpdateMode(info.update_mode);
        setCheckInterval(info.check_interval);
      }
    } catch (err) {
      console.error('Error loading update config:', err);
    }
  };

  const handleSaveUpdateConfig = async (e: Event) => {
    e.preventDefault();
    try {
      setSavingUpdate(true);
      await updateStore.updateConfig({
        mode: updateMode(),
        check_interval: checkInterval(),
      });
      setMessage({ type: 'success', text: 'Configuración de actualizaciones guardada' });
    } catch (err) {
      setMessage({ type: 'error', text: 'Error al guardar configuración de actualizaciones' });
    } finally {
      setSavingUpdate(false);
    }
  };

  const handleSaveToken = async (e: Event) => {
    e.preventDefault();
    if (!newToken().trim()) {
      setMessage({ type: 'error', text: 'El token no puede estar vacío' });
      return;
    }

    try {
      setSaving(true);
      await api.setGitHubToken(newToken());
      setTokenConfigured(true);
      setNewToken('');
      setMessage({ type: 'success', text: 'Token guardado correctamente' });
    } catch (err) {
      setMessage({ type: 'error', text: 'Error al guardar el token' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteToken = async () => {
    if (!confirm('¿Estás seguro de que deseas eliminar el token de GitHub?')) {
      return;
    }

    try {
      setSaving(true);
      await api.deleteGitHubToken();
      setTokenConfigured(false);
      setMessage({ type: 'success', text: 'Token eliminado correctamente' });
    } catch (err) {
      setMessage({ type: 'error', text: 'Error al eliminar el token' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center space-x-4">
          <button
            onClick={props.onBack}
            class="text-gray-500 hover:text-gray-700"
          >
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h1 class="text-2xl font-bold text-gray-900">Configuración</h1>
        </div>
      </div>

      {message() && (
        <div
          class={`mb-4 p-4 rounded-lg ${
            message()!.type === 'success'
              ? 'bg-green-50 text-green-700 border border-green-200'
              : 'bg-red-50 text-red-700 border border-red-200'
          }`}
        >
          {message()!.text}
        </div>
      )}

      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">Token de GitHub</h2>
        <p class="text-sm text-gray-600 mb-4">
          Configura un token de acceso personal de GitHub para poder clonar repositorios privados.
          El token necesita el permiso <code class="bg-gray-100 px-1 rounded">repo</code>.
        </p>

        {loading() ? (
          <div class="flex items-center justify-center py-8">
            <div class="w-8 h-8 border-4 border-nebula-500 border-t-transparent rounded-full animate-spin"></div>
          </div>
        ) : (
          <div>
            <div class="flex items-center mb-4">
              <span class="text-sm font-medium text-gray-700 mr-2">Estado:</span>
              {tokenConfigured() ? (
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Configurado
                </span>
              ) : (
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                  No configurado
                </span>
              )}
            </div>

            <form onSubmit={handleSaveToken} class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  {tokenConfigured() ? 'Actualizar token' : 'Token de GitHub'}
                </label>
                <input
                  type="password"
                  value={newToken()}
                  onInput={(e) => setNewToken(e.currentTarget.value)}
                  placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-nebula-500 focus:border-transparent"
                />
              </div>

              <div class="flex space-x-3">
                <button
                  type="submit"
                  disabled={saving() || !newToken().trim()}
                  class="px-4 py-2 bg-nebula-500 text-white rounded-lg hover:bg-nebula-600 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {saving() ? 'Guardando...' : tokenConfigured() ? 'Actualizar token' : 'Guardar token'}
                </button>

                {tokenConfigured() && (
                  <button
                    type="button"
                    onClick={handleDeleteToken}
                    disabled={saving()}
                    class="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Eliminar token
                  </button>
                )}
              </div>
            </form>

            <div class="mt-6 p-4 bg-gray-50 rounded-lg">
              <h3 class="text-sm font-medium text-gray-900 mb-2">¿Cómo obtener un token?</h3>
              <ol class="text-sm text-gray-600 list-decimal list-inside space-y-1">
                <li>Ve a GitHub → Settings → Developer settings → Personal access tokens</li>
                <li>Genera un nuevo token (classic) con el permiso "repo"</li>
                <li>Copia el token y pégalo aquí</li>
              </ol>
            </div>
          </div>
        )}
      </div>

      {/* Update Configuration Section */}
      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mt-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">Actualizaciones</h2>
        <p class="text-sm text-gray-600 mb-4">
          Configura cómo Nebula busca y aplica actualizaciones automáticas.
        </p>

        <form onSubmit={handleSaveUpdateConfig} class="space-y-4">
          <div>
            <label for="update-mode" class="block text-sm font-medium text-gray-700 mb-1">
              Modo de actualización
            </label>
            <select
              id="update-mode"
              value={updateMode()}
              onChange={(e) => setUpdateMode(e.currentTarget.value as 'auto' | 'notify' | 'disabled')}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-nebula-500 focus:border-transparent"
            >
              <option value="auto">Automático - Descarga e instala automáticamente</option>
              <option value="notify">Notificar - Avisa cuando hay actualizaciones</option>
              <option value="disabled">Desactivado - No buscar actualizaciones</option>
            </select>
          </div>

          <div>
            <label for="check-interval" class="block text-sm font-medium text-gray-700 mb-1">
              Intervalo de verificación (minutos)
            </label>
            <input
              id="check-interval"
              type="number"
              min="60"
              value={checkInterval()}
              onInput={(e) => setCheckInterval(parseInt(e.currentTarget.value) || 1440)}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-nebula-500 focus:border-transparent"
            />
            <p class="text-xs text-gray-500 mt-1">
              Mínimo 60 minutos. Por defecto: 1440 (24 horas)
            </p>
          </div>

          <button
            type="submit"
            disabled={savingUpdate()}
            class="px-4 py-2 bg-nebula-500 text-white rounded-lg hover:bg-nebula-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {savingUpdate() ? 'Guardando...' : 'Guardar configuración'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default Settings;
