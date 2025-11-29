# Contributing to Nebula

[English](#english) | [Español](#español)

---

## English

Thank you for your interest in contributing to Nebula! This document provides guidelines and instructions for contributing.

### How to Contribute

#### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/victalejo/nebula/issues)
2. If not, create a new issue with:
   - Clear and descriptive title
   - Steps to reproduce the bug
   - Expected behavior vs actual behavior
   - Environment details (OS, Docker version, etc.)

#### Suggesting Features

1. Check existing issues for similar suggestions
2. Create a new issue with:
   - Clear description of the feature
   - Use case and benefits
   - Possible implementation approach (optional)

#### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests and linting
5. Commit with clear messages
6. Push to your fork
7. Open a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/nebula.git
cd nebula

# Copy environment file
cp .env.example .env

# Backend development
make dev

# Frontend development (in another terminal)
make web-dev
```

### Code Style

#### Go (Backend)

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions small and focused
- Add comments for exported functions

#### TypeScript (Frontend)

- Use TypeScript strict mode
- Follow existing patterns in the codebase
- Use functional components with SolidJS

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb: "Add", "Fix", "Update", "Remove"
- Keep the first line under 72 characters

Example:
```
Add user authentication middleware

- Implement JWT validation
- Add auth routes
- Update API documentation
```

### Running Tests

```bash
# Run Go tests
make test

# Run frontend tests
cd web && npm test
```

---

## Español

¡Gracias por tu interés en contribuir a Nebula! Este documento proporciona guías e instrucciones para contribuir.

### Cómo Contribuir

#### Reportar Bugs

1. Verifica si el bug ya ha sido reportado en [Issues](https://github.com/victalejo/nebula/issues)
2. Si no, crea un nuevo issue con:
   - Título claro y descriptivo
   - Pasos para reproducir el bug
   - Comportamiento esperado vs comportamiento actual
   - Detalles del entorno (SO, versión de Docker, etc.)

#### Sugerir Funcionalidades

1. Revisa los issues existentes para sugerencias similares
2. Crea un nuevo issue con:
   - Descripción clara de la funcionalidad
   - Caso de uso y beneficios
   - Posible enfoque de implementación (opcional)

#### Pull Requests

1. Haz fork del repositorio
2. Crea una rama de feature: `git checkout -b feature/mi-feature`
3. Realiza tus cambios
4. Ejecuta tests y linting
5. Haz commit con mensajes claros
6. Push a tu fork
7. Abre un Pull Request

### Configuración de Desarrollo

```bash
# Clona tu fork
git clone https://github.com/TU_USUARIO/nebula.git
cd nebula

# Copia el archivo de entorno
cp .env.example .env

# Desarrollo del backend
make dev

# Desarrollo del frontend (en otra terminal)
make web-dev
```

### Estilo de Código

#### Go (Backend)

- Sigue las convenciones estándar de Go
- Usa `gofmt` para formatear
- Mantén las funciones pequeñas y enfocadas
- Agrega comentarios para funciones exportadas

#### TypeScript (Frontend)

- Usa TypeScript en modo estricto
- Sigue los patrones existentes en el código
- Usa componentes funcionales con SolidJS

### Mensajes de Commit

- Usa mensajes de commit claros y descriptivos
- Comienza con un verbo: "Add", "Fix", "Update", "Remove"
- Mantén la primera línea bajo 72 caracteres

Ejemplo:
```
Add user authentication middleware

- Implement JWT validation
- Add auth routes
- Update API documentation
```

### Ejecutar Tests

```bash
# Ejecutar tests de Go
make test

# Ejecutar tests del frontend
cd web && npm test
```
