import { Component, createSignal, For } from 'solid-js';

interface EditEnvVarsModalProps {
  environment: Record<string, string>;
  onSave: (env: Record<string, string>) => Promise<void>;
  onClose: () => void;
}

const EditEnvVarsModal: Component<EditEnvVarsModalProps> = (props) => {
  const [envVars, setEnvVars] = createSignal<Array<{ key: string; value: string }>>(
    Object.entries(props.environment || {}).map(([key, value]) => ({ key, value: String(value) }))
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
      await props.onSave(envVarsObj);
      props.onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Error al guardar');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between p-6 border-b sticky top-0 bg-white">
          <h2 class="text-lg font-semibold">Editar Variables de Entorno</h2>
          <button onClick={props.onClose} class="text-gray-400 hover:text-gray-600">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} class="p-6 space-y-4">
          <div>
            <div class="flex items-center justify-between mb-3">
              <p class="text-sm text-gray-500">
                Las variables se aplicaran en el proximo despliegue
              </p>
              <button
                type="button"
                onClick={addEnvVar}
                class="text-sm text-nebula-600 hover:text-nebula-700 font-medium"
              >
                + Agregar Variable
              </button>
            </div>

            <div class="space-y-2">
              <For each={envVars()} fallback={
                <div class="text-center py-8 text-gray-400 text-sm">
                  No hay variables configuradas
                </div>
              }>
                {(envVar, index) => (
                  <div class="flex space-x-2">
                    <input
                      type="text"
                      value={envVar.key}
                      onInput={(e) => updateEnvVar(index(), 'key', e.currentTarget.value)}
                      class="input flex-1 font-mono text-sm"
                      placeholder="NOMBRE_VARIABLE"
                    />
                    <input
                      type="text"
                      value={envVar.value}
                      onInput={(e) => updateEnvVar(index(), 'value', e.currentTarget.value)}
                      class="input flex-1 font-mono text-sm"
                      placeholder="valor"
                    />
                    <button
                      type="button"
                      onClick={() => removeEnvVar(index())}
                      class="p-2 text-gray-400 hover:text-red-500"
                      title="Eliminar"
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

          <div class="flex justify-end space-x-3 pt-4 border-t">
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
              {loading() ? 'Guardando...' : 'Guardar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default EditEnvVarsModal;
