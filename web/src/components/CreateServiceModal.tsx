import { Component, createSignal, Show } from 'solid-js';
import api, { CreateServiceRequest, ServiceType, BuilderType } from '../api/client';

interface CreateServiceModalProps {
  projectId: string;
  onClose: () => void;
  onCreated: () => void;
}

const CreateServiceModal: Component<CreateServiceModalProps> = (props) => {
  const [name, setName] = createSignal('');
  const [type, setType] = createSignal<ServiceType>('web');
  const [builder, setBuilder] = createSignal<BuilderType>('docker_image');
  const [dockerImage, setDockerImage] = createSignal('');
  const [gitRepo, setGitRepo] = createSignal('');
  const [gitBranch, setGitBranch] = createSignal('main');
  const [subdirectory, setSubdirectory] = createSignal('');
  const [port, setPort] = createSignal(8080);
  const [command, setCommand] = createSignal('');
  const [databaseType, setDatabaseType] = createSignal('postgres');
  const [databaseVersion, setDatabaseVersion] = createSignal('16');
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const data: CreateServiceRequest = {
      name: name(),
      type: type(),
      builder: type() === 'database' ? undefined : builder(),
      port: type() !== 'database' ? port() : undefined,
    };

    if (type() === 'database') {
      data.database_type = databaseType();
      data.database_version = databaseVersion();
    } else if (builder() === 'docker_image') {
      data.docker_image = dockerImage();
    } else {
      data.git_repo = gitRepo();
      data.git_branch = gitBranch();
      if (subdirectory()) data.subdirectory = subdirectory();
    }

    if (command()) data.command = command();

    try {
      await api.createService(props.projectId, data);
      props.onCreated();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create service');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between p-6 border-b sticky top-0 bg-white">
          <h2 class="text-lg font-semibold">Crear Servicio</h2>
          <button onClick={props.onClose} class="text-gray-400 hover:text-gray-600">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} class="p-6 space-y-4">
          <div>
            <label class="label">Nombre del Servicio *</label>
            <input
              type="text"
              value={name()}
              onInput={(e) => setName(e.currentTarget.value)}
              class="input"
              placeholder="backend"
              pattern="[a-z0-9-]+"
              required
            />
            <p class="text-xs text-gray-500 mt-1">Solo letras minusculas, numeros y guiones</p>
          </div>

          <div>
            <label class="label">Tipo de Servicio</label>
            <select
              value={type()}
              onChange={(e) => setType(e.currentTarget.value as ServiceType)}
              class="input"
            >
              <option value="web">Web (HTTP)</option>
              <option value="worker">Worker (Background)</option>
              <option value="cron">Cron (Scheduled)</option>
              <option value="database">Base de Datos</option>
            </select>
          </div>

          <Show when={type() === 'database'}>
            <div>
              <label class="label">Tipo de Base de Datos</label>
              <select
                value={databaseType()}
                onChange={(e) => setDatabaseType(e.currentTarget.value)}
                class="input"
              >
                <option value="postgres">PostgreSQL</option>
                <option value="mysql">MySQL</option>
                <option value="redis">Redis</option>
                <option value="mongodb">MongoDB</option>
              </select>
            </div>
            <div>
              <label class="label">Version</label>
              <input
                type="text"
                value={databaseVersion()}
                onInput={(e) => setDatabaseVersion(e.currentTarget.value)}
                class="input"
                placeholder="16"
              />
            </div>
          </Show>

          <Show when={type() !== 'database'}>
            <div>
              <label class="label">Builder</label>
              <select
                value={builder()}
                onChange={(e) => setBuilder(e.currentTarget.value as BuilderType)}
                class="input"
              >
                <option value="docker_image">Imagen Docker</option>
                <option value="nixpacks">Nixpacks (Auto-detect)</option>
                <option value="dockerfile">Dockerfile</option>
                <option value="buildpacks">Buildpacks</option>
              </select>
            </div>

            <Show when={builder() === 'docker_image'}>
              <div>
                <label class="label">Imagen Docker *</label>
                <input
                  type="text"
                  value={dockerImage()}
                  onInput={(e) => setDockerImage(e.currentTarget.value)}
                  class="input"
                  placeholder="nginx:latest"
                  required={builder() === 'docker_image'}
                />
              </div>
            </Show>

            <Show when={builder() !== 'docker_image'}>
              <div>
                <label class="label">Repositorio Git *</label>
                <input
                  type="url"
                  value={gitRepo()}
                  onInput={(e) => setGitRepo(e.currentTarget.value)}
                  class="input"
                  placeholder="https://github.com/user/repo"
                  required={builder() !== 'docker_image'}
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
              <div>
                <label class="label">Subdirectorio (opcional)</label>
                <input
                  type="text"
                  value={subdirectory()}
                  onInput={(e) => setSubdirectory(e.currentTarget.value)}
                  class="input"
                  placeholder="apps/api"
                />
                <p class="text-xs text-gray-500 mt-1">Para monorepos</p>
              </div>
            </Show>

            <div>
              <label class="label">Puerto</label>
              <input
                type="number"
                value={port()}
                onInput={(e) => setPort(parseInt(e.currentTarget.value) || 8080)}
                class="input"
                min="1"
                max="65535"
              />
            </div>

            <div>
              <label class="label">Comando (opcional)</label>
              <input
                type="text"
                value={command()}
                onInput={(e) => setCommand(e.currentTarget.value)}
                class="input"
                placeholder="npm start"
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
              {loading() ? 'Creando...' : 'Crear Servicio'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateServiceModal;
