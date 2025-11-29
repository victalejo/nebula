import { createSignal, createRoot } from 'solid-js';
import api from '../api/client';

function createAuthStore() {
  const [isAuthenticated, setIsAuthenticated] = createSignal(!!api.getToken());
  const [user, setUser] = createSignal<{ username: string } | null>(null);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  async function login(username: string, password: string) {
    setLoading(true);
    setError(null);
    try {
      await api.login(username, password);
      setIsAuthenticated(true);
      await fetchUser();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Error de inicio de sesión');
      throw e;
    } finally {
      setLoading(false);
    }
  }

  function logout() {
    api.clearToken();
    setIsAuthenticated(false);
    setUser(null);
  }

  async function fetchUser() {
    if (!api.getToken()) return;
    try {
      const userData = await api.getUser();
      setUser(userData);
    } catch {
      logout();
    }
  }

  async function checkAuth() {
    if (api.getToken()) {
      await fetchUser();
    }
  }

  return {
    isAuthenticated,
    user,
    loading,
    error,
    login,
    logout,
    checkAuth,
  };
}

export const authStore = createRoot(createAuthStore);
