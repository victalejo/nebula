import { Component, Show, onMount } from 'solid-js';
import { updateStore } from '../stores/update';

const UpdateBanner: Component = () => {
  onMount(() => {
    updateStore.fetchSystemInfo();
    updateStore.fetchUpdateStatus();
  });

  const hasUpdate = () => {
    const info = updateStore.updateInfo();
    return info?.available && !updateStore.dismissed();
  };

  const isDownloading = () => {
    const status = updateStore.status();
    return status.state === 'downloading';
  };

  const isReady = () => {
    const status = updateStore.status();
    return status.state === 'ready';
  };

  const isApplying = () => {
    const status = updateStore.status();
    return status.state === 'applying' || status.state === 'restarting';
  };

  const showBanner = () => hasUpdate() || isDownloading() || isReady() || isApplying();

  return (
    <Show when={showBanner()}>
      <div class="bg-gradient-to-r from-cyan-500 to-cyan-600 text-white px-4 py-3 shadow-lg">
        <div class="max-w-7xl mx-auto flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>

            <Show when={isApplying()}>
              <div class="flex items-center space-x-2">
                <div class="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent"></div>
                <span>Aplicando actualizacion... El servidor se reiniciara pronto.</span>
              </div>
            </Show>

            <Show when={isDownloading()}>
              <div class="flex items-center space-x-3">
                <span>Descargando actualizacion...</span>
                <div class="w-32 bg-white/30 rounded-full h-2">
                  <div
                    class="bg-white rounded-full h-2 transition-all duration-300"
                    style={{ width: `${updateStore.status().progress}%` }}
                  ></div>
                </div>
                <span class="text-sm">{updateStore.status().progress.toFixed(0)}%</span>
              </div>
            </Show>

            <Show when={isReady()}>
              <span>Actualizacion lista. El servidor se reiniciara automaticamente.</span>
            </Show>

            <Show when={hasUpdate() && !isDownloading() && !isReady() && !isApplying()}>
              <span>
                Nueva version disponible: <strong>v{updateStore.updateInfo()?.latest_version}</strong>
                <span class="text-cyan-100 ml-2">(actual: v{updateStore.updateInfo()?.current_version})</span>
              </span>
            </Show>
          </div>

          <div class="flex items-center space-x-3">
            <Show when={hasUpdate() && !isDownloading() && !isReady() && !isApplying()}>
              <button
                onClick={() => updateStore.applyUpdate()}
                disabled={updateStore.loading()}
                class="bg-white text-cyan-600 px-4 py-1.5 rounded-md text-sm font-medium hover:bg-cyan-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {updateStore.loading() ? 'Actualizando...' : 'Actualizar ahora'}
              </button>
              <button
                onClick={() => updateStore.dismiss()}
                class="text-white/80 hover:text-white transition-colors p-1"
                title="Descartar"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </Show>
          </div>
        </div>

        <Show when={updateStore.error()}>
          <div class="mt-2 text-red-100 text-sm">
            Error: {updateStore.error()}
          </div>
        </Show>
      </div>
    </Show>
  );
};

export default UpdateBanner;
