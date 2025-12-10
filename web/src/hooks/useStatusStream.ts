import { onMount, onCleanup, createSignal } from 'solid-js';
import api, { StatusEvent } from '../api/client';

/**
 * Hook to subscribe to real-time status updates for a project via SSE.
 * Automatically handles connection, reconnection, and cleanup.
 */
export function useProjectStatusStream(
  projectId: () => string | undefined,
  onStatusChange: (event: StatusEvent) => void
) {
  const [connected, setConnected] = createSignal(false);
  let eventSource: EventSource | null = null;
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  const connect = () => {
    const id = projectId();
    if (!id) return;

    // Close existing connection
    if (eventSource) {
      eventSource.close();
    }

    eventSource = api.streamProjectStatus(id);

    eventSource.onopen = () => {
      setConnected(true);
    };

    eventSource.addEventListener('connected', () => {
      setConnected(true);
    });

    eventSource.addEventListener('status', (e: MessageEvent) => {
      try {
        const event: StatusEvent = JSON.parse(e.data);
        onStatusChange(event);
      } catch (err) {
        console.error('Failed to parse status event:', err);
      }
    });

    eventSource.onerror = () => {
      setConnected(false);
      eventSource?.close();
      // Reconnect after 5 seconds
      reconnectTimeout = setTimeout(connect, 5000);
    };
  };

  onMount(() => {
    connect();
  });

  onCleanup(() => {
    if (eventSource) {
      eventSource.close();
    }
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }
  });

  return { connected };
}

/**
 * Hook to subscribe to global status updates (all projects) via SSE.
 */
export function useGlobalStatusStream(
  onStatusChange: (event: StatusEvent) => void
) {
  const [connected, setConnected] = createSignal(false);
  let eventSource: EventSource | null = null;
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  const connect = () => {
    if (eventSource) {
      eventSource.close();
    }

    eventSource = api.streamGlobalStatus();

    eventSource.onopen = () => {
      setConnected(true);
    };

    eventSource.addEventListener('connected', () => {
      setConnected(true);
    });

    eventSource.addEventListener('status', (e: MessageEvent) => {
      try {
        const event: StatusEvent = JSON.parse(e.data);
        onStatusChange(event);
      } catch (err) {
        console.error('Failed to parse status event:', err);
      }
    });

    eventSource.onerror = () => {
      setConnected(false);
      eventSource?.close();
      reconnectTimeout = setTimeout(connect, 5000);
    };
  };

  onMount(() => {
    connect();
  });

  onCleanup(() => {
    if (eventSource) {
      eventSource.close();
    }
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }
  });

  return { connected };
}
