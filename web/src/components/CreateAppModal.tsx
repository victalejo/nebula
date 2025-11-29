import { Component, createSignal, Show } from 'solid-js';
import api, { CreateAppRequest } from '../api/client';

interface CreateAppModalProps {
  onClose: () => void;
  onCreated: () => void;
}

const CreateAppModal: Component<CreateAppModalProps> = (props) => {
  const [name, setName] = createSignal('');
  const [mode, setMode] = createSignal<'git' | 'docker_image' | 'docker_compose'>('docker_image');
  const [domain, setDomain] = createSignal('');
  const [image, setImage] = createSignal('');
  const [gitRepo, setGitRepo] = createSignal('');
  const [gitBranch, setGitBranch] = createSignal('main');
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const data: CreateAppRequest = {
      name: name(),
      deployment_mode: mode(),
      domain: domain() || undefined,
    };

    if (mode() === 'docker_image') {
      data.docker_image = image();
    } else if (mode() === 'git') {
      data.git_repo = gitRepo();
      data.git_branch = gitBranch();
    }

    try {
      await api.createApp(data);
      props.onCreated();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create app');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md">
        <div class="flex items-center justify-between p-6 border-b">
          <h2 class="text-lg font-semibold">Crear Aplicación</h2>
          <button onClick={props.onClose} class="text-gray-400 hover:text-gray-600">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} class="p-6 space-y-4">
          <div>
            <label class="label">Nombre de App</label>
            <input
              type="text"
              value={name()}
              onInput={(e) => setName(e.currentTarget.value)}
              class="input"
              placeholder="mi-app"
              pattern="[a-z0-9-]+"
              required
            />
            <p class="text-xs text-gray-500 mt-1">Solo letras minúsculas, números y guiones</p>
          </div>

          <div>
            <label class="label">Modo de Despliegue</label>
            <select
              value={mode()}
              onChange={(e) => setMode(e.currentTarget.value as any)}
              class="input"
            >
              <option value="docker_image">Imagen Docker</option>
              <option value="git">Repositorio Git</option>
              <option value="docker_compose">Docker Compose</option>
            </select>
          </div>

          <div>
            <label class="label">Dominio (opcional)</label>
            <input
              type="text"
              value={domain()}
              onInput={(e) => setDomain(e.currentTarget.value)}
              class="input"
              placeholder="miapp.ejemplo.com"
            />
          </div>

          <Show when={mode() === 'docker_image'}>
            <div>
              <label class="label">Imagen Docker</label>
              <input
                type="text"
                value={image()}
                onInput={(e) => setImage(e.currentTarget.value)}
                class="input"
                placeholder="nginx:latest"
              />
            </div>
          </Show>

          <Show when={mode() === 'git'}>
            <div>
              <label class="label">URL del Repositorio Git</label>
              <input
                type="url"
                value={gitRepo()}
                onInput={(e) => setGitRepo(e.currentTarget.value)}
                class="input"
                placeholder="https://github.com/usuario/repo"
              />
            </div>
            <div>
              <label class="label">Rama</label>
              <input
                type="text"
                value={gitBranch()}
                onInput={(e) => setGitBranch(e.currentTarget.value)}
                class="input"
                placeholder="main"
              />
            </div>
          </Show>

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
              {loading() ? 'Creando...' : 'Crear App'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateAppModal;
