import { Component, createSignal } from 'solid-js';
import api, { CreateAppRequest } from '../api/client';

interface CreateAppModalProps {
  onClose: () => void;
  onCreated: () => void;
}

const CreateAppModal: Component<CreateAppModalProps> = (props) => {
  const [name, setName] = createSignal('');
  const [displayName, setDisplayName] = createSignal('');
  const [description, setDescription] = createSignal('');
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const data: CreateAppRequest = {
      name: name(),
      display_name: displayName() || undefined,
      description: description() || undefined,
    };

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
            <label class="label">Nombre de App *</label>
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
            <label class="label">Nombre para mostrar (opcional)</label>
            <input
              type="text"
              value={displayName()}
              onInput={(e) => setDisplayName(e.currentTarget.value)}
              class="input"
              placeholder="Mi Aplicación"
            />
          </div>

          <div>
            <label class="label">Descripción (opcional)</label>
            <textarea
              value={description()}
              onInput={(e) => setDescription(e.currentTarget.value)}
              class="input"
              placeholder="Descripción de la aplicación..."
              rows={2}
            />
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
              {loading() ? 'Creando...' : 'Crear App'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateAppModal;
