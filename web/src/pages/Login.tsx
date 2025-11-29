import { Component, createSignal } from 'solid-js';
import { authStore } from '../stores/auth';

const Login: Component = () => {
  const [username, setUsername] = createSignal('');
  const [password, setPassword] = createSignal('');

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    try {
      await authStore.login(username(), password());
    } catch {
      // Error is handled in store
    }
  };

  return (
    <div class="min-h-screen bg-gradient-to-br from-nebula-500 to-purple-600 flex items-center justify-center p-4">
      <div class="bg-white rounded-2xl shadow-xl w-full max-w-md p-8">
        <div class="text-center mb-8">
          <svg class="w-16 h-16 text-nebula-500 mx-auto" viewBox="0 0 24 24" fill="currentColor">
            <circle cx="12" cy="12" r="10" opacity="0.2" />
            <circle cx="12" cy="12" r="6" opacity="0.4" />
            <circle cx="12" cy="12" r="3" />
          </svg>
          <h1 class="text-2xl font-bold text-gray-900 mt-4">Nebula</h1>
          <p class="text-gray-600">Lightweight PaaS</p>
        </div>

        <form onSubmit={handleSubmit} class="space-y-4">
          <div>
            <label class="label">Username</label>
            <input
              type="text"
              value={username()}
              onInput={(e) => setUsername(e.currentTarget.value)}
              class="input"
              placeholder="admin"
              required
            />
          </div>

          <div>
            <label class="label">Password</label>
            <input
              type="password"
              value={password()}
              onInput={(e) => setPassword(e.currentTarget.value)}
              class="input"
              placeholder="********"
              required
            />
          </div>

          {authStore.error() && (
            <div class="bg-red-50 text-red-600 px-4 py-2 rounded-lg text-sm">
              {authStore.error()}
            </div>
          )}

          <button
            type="submit"
            disabled={authStore.loading()}
            class="btn btn-primary w-full disabled:opacity-50"
          >
            {authStore.loading() ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default Login;
