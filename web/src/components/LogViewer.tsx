import { Component, createSignal, onMount, onCleanup, For } from 'solid-js';
import api from '../api/client';

interface LogViewerProps {
  appName: string;
}

interface LogLine {
  id: number;
  timestamp: string;
  message: string;
  level?: 'info' | 'warn' | 'error';
}

const LogViewer: Component<LogViewerProps> = (props) => {
  const [logs, setLogs] = createSignal<LogLine[]>([]);
  const [connected, setConnected] = createSignal(false);
  const [autoScroll, setAutoScroll] = createSignal(true);
  const [filter, setFilter] = createSignal('');
  let logContainer: HTMLDivElement | undefined;
  let eventSource: EventSource | null = null;
  let logId = 0;

  const connect = () => {
    if (eventSource) {
      eventSource.close();
    }

    eventSource = api.streamLogs(props.appName, { follow: true, tail: 100 });

    eventSource.onopen = () => {
      setConnected(true);
    };

    eventSource.onmessage = (event) => {
      const line = event.data;
      const level = detectLogLevel(line);

      setLogs((prev) => [
        ...prev.slice(-999), // Keep last 1000 lines
        {
          id: ++logId,
          timestamp: new Date().toISOString(),
          message: line,
          level,
        },
      ]);

      if (autoScroll() && logContainer) {
        setTimeout(() => {
          logContainer!.scrollTop = logContainer!.scrollHeight;
        }, 0);
      }
    };

    eventSource.onerror = () => {
      setConnected(false);
      // Retry connection after 5 seconds
      setTimeout(connect, 5000);
    };
  };

  const detectLogLevel = (line: string): 'info' | 'warn' | 'error' | undefined => {
    const lower = line.toLowerCase();
    if (lower.includes('error') || lower.includes('fatal') || lower.includes('panic')) {
      return 'error';
    }
    if (lower.includes('warn') || lower.includes('warning')) {
      return 'warn';
    }
    if (lower.includes('info')) {
      return 'info';
    }
    return undefined;
  };

  const getLogColor = (level?: string) => {
    switch (level) {
      case 'error':
        return 'text-red-400';
      case 'warn':
        return 'text-yellow-400';
      case 'info':
        return 'text-blue-400';
      default:
        return 'text-gray-300';
    }
  };

  const filteredLogs = () => {
    const f = filter().toLowerCase();
    if (!f) return logs();
    return logs().filter((log) => log.message.toLowerCase().includes(f));
  };

  const clearLogs = () => {
    setLogs([]);
  };

  onMount(connect);

  onCleanup(() => {
    if (eventSource) {
      eventSource.close();
    }
  });

  return (
    <div class="card p-0 overflow-hidden">
      {/* Toolbar */}
      <div class="flex items-center justify-between p-3 border-b bg-gray-50">
        <div class="flex items-center space-x-3">
          <div class="flex items-center space-x-2">
            <span
              class={`w-2 h-2 rounded-full ${
                connected() ? 'bg-green-500' : 'bg-red-500'
              }`}
            ></span>
            <span class="text-sm text-gray-600">
              {connected() ? 'Conectado' : 'Desconectado'}
            </span>
          </div>
          <span class="text-sm text-gray-400">|</span>
          <span class="text-sm text-gray-600">{logs().length} líneas</span>
        </div>

        <div class="flex items-center space-x-3">
          <input
            type="text"
            value={filter()}
            onInput={(e) => setFilter(e.currentTarget.value)}
            class="px-3 py-1 text-sm border rounded-lg w-48"
            placeholder="Filtrar registros..."
          />
          <label class="flex items-center space-x-2 text-sm text-gray-600">
            <input
              type="checkbox"
              checked={autoScroll()}
              onChange={(e) => setAutoScroll(e.currentTarget.checked)}
              class="rounded"
            />
            <span>Auto-scroll</span>
          </label>
          <button
            onClick={clearLogs}
            class="text-sm text-gray-500 hover:text-gray-700"
          >
            Limpiar
          </button>
        </div>
      </div>

      {/* Log Output */}
      <div
        ref={logContainer}
        class="bg-gray-900 p-4 h-96 overflow-y-auto font-mono text-sm"
      >
        <For each={filteredLogs()}>
          {(log) => (
            <div class={`${getLogColor(log.level)} hover:bg-gray-800 px-1 -mx-1`}>
              <span class="text-gray-500 select-none mr-2">
                {new Date(log.timestamp).toLocaleTimeString()}
              </span>
              {log.message}
            </div>
          )}
        </For>

        {logs().length === 0 && (
          <div class="text-gray-500 text-center py-8">
            Esperando registros...
          </div>
        )}
      </div>
    </div>
  );
};

export default LogViewer;
