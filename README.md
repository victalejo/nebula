# Nebula

A self-hosted Platform as a Service (PaaS) for deploying and managing applications on your own infrastructure.

[English](#english) | [Español](#español)

---

## English

### Features

- **Multiple Deployment Strategies** - Deploy from Docker images, Git repositories, or Docker Compose files
- **Blue-Green Deployments** - Zero-downtime deployments with automatic traffic switching
- **Automatic SSL/TLS** - Built-in Caddy proxy with automatic HTTPS certificates
- **Modern Dashboard** - Clean web interface built with SolidJS
- **Environment Variables** - Secure configuration management per application
- **Custom Domains** - Assign custom domains to your applications
- **Real-time Logs** - Stream application logs directly from the dashboard

### Requirements

- Docker
- Docker Compose

### Quick Start

```bash
# Clone the repository
git clone https://github.com/victalejo/nebula.git
cd nebula

# Copy environment file
cp .env.example .env

# Start Nebula
docker compose up -d
```

Access the dashboard at `http://localhost`

### Local Development

```bash
# Build everything
make build

# Run the server
make run

# Development mode with hot-reload
make dev

# Frontend development
make web-dev
```

### Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go, Gin Framework |
| Frontend | SolidJS, TypeScript, Tailwind CSS |
| Database | SQLite |
| Proxy | Caddy |
| Container | Docker |

### Project Structure

```
nebula/
├── cmd/                  # Application entrypoints
├── internal/             # Private application code
│   ├── api/             # REST API handlers
│   ├── service/         # Business logic
│   ├── container/       # Docker integration
│   ├── deployer/        # Deployment strategies
│   └── proxy/           # Caddy proxy management
├── web/                  # SolidJS frontend
└── docker-compose.yml
```

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute.

### License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Español

### Características

- **Múltiples Estrategias de Despliegue** - Despliega desde imágenes Docker, repositorios Git o archivos Docker Compose
- **Despliegues Blue-Green** - Despliegues sin tiempo de inactividad con cambio automático de tráfico
- **SSL/TLS Automático** - Proxy Caddy integrado con certificados HTTPS automáticos
- **Dashboard Moderno** - Interfaz web limpia construida con SolidJS
- **Variables de Entorno** - Gestión segura de configuración por aplicación
- **Dominios Personalizados** - Asigna dominios personalizados a tus aplicaciones
- **Logs en Tiempo Real** - Transmite logs de aplicaciones directamente desde el dashboard

### Requisitos

- Docker
- Docker Compose

### Inicio Rápido

```bash
# Clonar el repositorio
git clone https://github.com/victalejo/nebula.git
cd nebula

# Copiar archivo de entorno
cp .env.example .env

# Iniciar Nebula
docker compose up -d
```

Accede al dashboard en `http://localhost`

### Desarrollo Local

```bash
# Compilar todo
make build

# Ejecutar el servidor
make run

# Modo desarrollo con hot-reload
make dev

# Desarrollo del frontend
make web-dev
```

### Stack Tecnológico

| Componente | Tecnología |
|------------|------------|
| Backend | Go, Gin Framework |
| Frontend | SolidJS, TypeScript, Tailwind CSS |
| Base de Datos | SQLite |
| Proxy | Caddy |
| Contenedor | Docker |

### Estructura del Proyecto

```
nebula/
├── cmd/                  # Puntos de entrada
├── internal/             # Código privado
│   ├── api/             # Handlers REST API
│   ├── service/         # Lógica de negocio
│   ├── container/       # Integración Docker
│   ├── deployer/        # Estrategias de despliegue
│   └── proxy/           # Gestión de proxy Caddy
├── web/                  # Frontend SolidJS
└── docker-compose.yml
```

### Contribuir

Consulta [CONTRIBUTING.md](CONTRIBUTING.md) para ver las guías de cómo contribuir.

### Licencia

Este proyecto está licenciado bajo la Licencia MIT - consulta el archivo [LICENSE](LICENSE) para más detalles.
