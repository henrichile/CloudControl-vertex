# Cloud Control

> Plataforma inteligente de gestión y orquestación de contenedores con AIOps integrado.

Cloud Control es un monorepo que combina un backend Go de alto rendimiento, una CLI robusta y un WebAdmin moderno para gestionar contenedores Docker con capacidades de inteligencia artificial a través de Ollama.

---

## Arquitectura

```
CloudControl/
├── backend/          # API REST en Go + gestión Docker + AIOps
├── cli/              # cloudctl — CLI en Go con Cobra
└── frontend/         # WebAdmin en React + TypeScript + Tailwind
```

### Stack Tecnológico

| Capa       | Tecnología                                      |
|------------|------------------------------------------------|
| Backend    | Go 1.22, Gin, GORM, Docker SDK, JWT            |
| Base Datos | SQLite (dev) / PostgreSQL (prod)                |
| Frontend   | React 18, TypeScript, Vite, Tailwind, Recharts |
| CLI        | Go + Cobra + Viper                             |
| IA Local   | Ollama (llama3 / mistral)                      |
| Contenedor | Docker Engine API v1.45+                        |

---

## Características

### Gestión de Contenedores
- Listar, inspeccionar, iniciar, detener y eliminar contenedores
- Build de imágenes y gestión de Docker Compose
- Stream de logs en tiempo real vía WebSocket
- Métricas de CPU, RAM y red en vivo

### Generación Automática de Proyectos
Plantillas integradas para stacks comunes:
- `MERN` — MongoDB + Express + React + Node
- `LAMP` — Linux + Apache + MySQL + PHP
- `laravel-redis` — Laravel + Redis + MySQL
- `fastapi-postgres` — FastAPI + PostgreSQL
- `data-science` — Jupyter + Pandas + PostgreSQL
- `nextjs-prisma` — Next.js + Prisma + PostgreSQL

### AIOps (Orquestación Inteligente)
- Análisis de métricas en tiempo real con Ollama
- Sugerencias de escalado vertical (CPU/RAM limits)
- Aplicación automática de límites de recursos

### Seguridad Proactiva
- Auditoría de `docker-compose.yml`, `.env` y configs de Nginx
- Detección de secrets expuestos, puertos peligrosos, imágenes sin fijar
- Generación de parches y reglas de firewall via IA

---

## Inicio Rápido

### Prerrequisitos
- Go 1.22+
- Node.js 20+
- Docker Engine
- Ollama con modelo `llama3` o `mistral`

### Instalación

```bash
# Clonar el repositorio
git clone https://github.com/etasoft/cloudcontrol
cd cloudcontrol

# Construir todo con Make
make install

# Iniciar backend
make dev-backend

# Iniciar frontend (otra terminal)
make dev-frontend

# Usar la CLI
cloudctl --help
```

### Variables de Entorno

```bash
# backend/.env
PORT=8080
DATABASE_PATH=./cloudcontrol.db
DOCKER_HOST=unix:///var/run/docker.sock
OLLAMA_HOST=http://localhost:11434
OLLAMA_MODEL=llama3
JWT_SECRET=change-me-in-production
```

---

## CLI — cloudctl

```bash
# Contenedores
cloudctl containers list
cloudctl containers start <nombre>
cloudctl containers stop <nombre>
cloudctl containers logs <nombre> --follow

# Proyectos
cloudctl projects create --name mi-app --stack MERN
cloudctl projects up --name mi-app
cloudctl projects down --name mi-app

# AIOps
cloudctl aiops analyze <contenedor>      # Análisis de métricas con IA
cloudctl aiops audit --file docker-compose.yml  # Auditoría de seguridad
```

---

## API REST

```
GET    /api/v1/health
GET    /api/v1/containers
GET    /api/v1/containers/:id
POST   /api/v1/containers/:id/start
POST   /api/v1/containers/:id/stop
GET    /api/v1/containers/:id/logs
GET    /api/v1/containers/:id/stats      (WebSocket)

GET    /api/v1/projects
POST   /api/v1/projects
POST   /api/v1/projects/:id/up
POST   /api/v1/projects/:id/down

POST   /api/v1/aiops/analyze
POST   /api/v1/aiops/audit
```

---

## Desarrollo

```bash
make test        # Ejecutar tests
make lint        # Linting (golangci-lint + eslint)
make build       # Build de producción
make docker-dev  # Levantar entorno de desarrollo completo
```

---

## Licencia

MIT © Etasoft
