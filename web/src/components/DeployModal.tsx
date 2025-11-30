import { Component, createSignal, Show, For } from 'solid-js';
import api, { App, DeployImageRequest, DeployGitRequest } from '../api/client';

interface DeployModalProps {
  app: App;
  onClose: () => void;
  onDeployed: () => void;
}

const DeployModal: Component<DeployModalProps> = (props) => {
  const [image, setImage] = createSignal(props.app.docker_image || '');
  const [tag, setTag] = createSignal('latest');
  const [port, setPort] = createSignal(80);
  const [envVars, setEnvVars] = createSignal<Array<{ key: string; value: string }>>(
    Object.entries(props.app.environment || {}).map(([key, value]) => ({ key, value: String(value) }))
  );
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const addEnvVar = () => {
    setEnvVars([...envVars(), { key: '', value: '' }]);
  };

  const removeEnvVar = (index: number) => {
    setEnvVars(envVars().filter((_, i) => i !== index));
  };

  const updateEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...envVars()];
    updated[index][field] = value;
    setEnvVars(updated);
  };

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const envVarsObj: Record<string, string> = {};
    for (const { key, value } of envVars()) {
      if (key.trim()) {
        envVarsObj[key.trim()] = value;
      }
    }

    try {
      if (props.app.deployment_mode === 'docker_image') {
        // Combine image and tag
        const fullImage = tag() ? `${image()}:${tag()}` : image();

        const data: DeployImageRequest = {
          image: fullImage,
          port: port(),
          environment: Object.keys(envVarsObj).length > 0 ? envVarsObj : undefined,
        };
        await api.deployImage(props.app.name, data);
      } else if (props.app.deployment_mode === 'git') {
        const data: DeployGitRequest = {
          branch: props.app.git_branch || 'main',
          environment: Object.keys(envVarsObj).length > 0 ? envVarsObj : undefined,
        };
        await api.deployGit(props.app.name, data);
      }
      props.onDeployed();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Deployment failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between p-6 border-b sticky top-0 bg-white">
          <h2 class="text-lg font-semibold">Desplegar {props.app.name}</h2>
          <button onClick={props.onClose} class="text-gray-400 hover:text-gray-600">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} class="p-6 space-y-4">
          <Show when={props.app.deployment_mode === 'docker_image'}>
            <div>
              <label class="label">Imagen Docker</label>
              <input
                type="text"
                value={image()}
                onInput={(e) => setImage(e.currentTarget.value)}
                class="input"
                placeholder="nginx"
                required
              />
            </div>

            <div>
              <label class="label">Etiqueta</label>
              <input
                type="text"
                value={tag()}
                onInput={(e) => setTag(e.currentTarget.value)}
                class="input"
                placeholder="latest"
              />
            </div>

            <div>
              <label class="label">Puerto del Contenedor</label>
              <input
                type="number"
                value={port()}
                onInput={(e) => setPort(parseInt(e.currentTarget.value) || 80)}
                class="input"
                placeholder="80"
                min="1"
                max="65535"
                required
              />
              <p class="text-xs text-gray-500 mt-1">El puerto en el que escucha tu aplicación dentro del contenedor</p>
            </div>
          </Show>

          <Show when={props.app.deployment_mode === 'git'}>
            <div class="bg-blue-50 text-blue-700 px-4 py-3 rounded-lg text-sm">
              Esto descargará el código más reciente de <strong>{props.app.git_repo}</strong>
              (rama: {props.app.git_branch || 'main'}) y reconstruirá la aplicación.
            </div>
          </Show>

          {/* Environment Variables */}
          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="label mb-0">Variables de Entorno</label>
              <button
                type="button"
                onClick={addEnvVar}
                class="text-sm text-nebula-600 hover:text-nebula-700"
              >
                + Agregar Variable
              </button>
            </div>

            <div class="space-y-2">
              <For each={envVars()}>
                {(envVar, index) => (
                  <div class="flex space-x-2">
                    <input
                      type="text"
                      value={envVar.key}
                      onInput={(e) => updateEnvVar(index(), 'key', e.currentTarget.value)}
                      class="input flex-1"
                      placeholder="CLAVE"
                    />
                    <input
                      type="text"
                      value={envVar.value}
                      onInput={(e) => updateEnvVar(index(), 'value', e.currentTarget.value)}
                      class="input flex-1"
                      placeholder="valor"
                    />
                    <button
                      type="button"
                      onClick={() => removeEnvVar(index())}
                      class="p-2 text-gray-400 hover:text-red-500"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                )}
              </For>
            </div>
          </div>

          {error() && (
            <div class="bg-red-50 text-red-600 px-4 py-2 rounded-lg text-sm">
              {error()}
            </div>
          )}

          <div class="flex justify-end space-x-3 pt-4">
            <button
              type="button"
              onClick={props.onClose}
              class="btn btn-secondary"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={loading()}
              class="btn btn-primary disabled:opacity-50"
            >
              {loading() ? 'Desplegando...' : 'Desplegar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default DeployModal;
