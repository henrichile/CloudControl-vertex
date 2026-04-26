# Cloud Control

> Plataforma inteligente de gestión y orquestación de contenedores con AIOps integrado.

Cloud Control combina un backend Go de alto rendimiento, una CLI robusta y un WebAdmin moderno para gestionar contenedores Docker con inteligencia artificial local a través de Ollama.

---

## Arquitectura

```
CloudControl/
├── backend/           # API REST en Go + gestión Docker + AIOps
├── cli/               # cloudctl — CLI en Go con Cobra
├── frontend/          # WebAdmin en React + TypeScript + Tailwind
└── infrastructure/    # Traefik v3 — reverse proxy global
```

### Stack tecnológico

| Capa | Tecnología |
|---|---|
| Backend | Go 1.22, Gin, GORM, Docker SDK, JWT |
| Base de datos | SQLite (dev) / PostgreSQL (prod) |
| Frontend | React 18, TypeScript, Vite, Tailwind, Recharts |
| CLI | Go + Cobra + Viper |
| IA local | Ollama (llama3 / mistral / gemma2 / qwen2) |
| Reverse proxy | Traefik v3 con TLS automático (Let's Encrypt) |
| Contenedores | Docker Engine API v1.45+ |

---

## Instalación en producción

### Requisitos del servidor

| Recurso | Mínimo | Recomendado |
|---|---|---|
| OS | Ubuntu 22.04, Debian 11, AlmaLinux 8 | Ubuntu 24.04, AlmaLinux 10 |
| CPU | 2 vCPU | 4 vCPU |
| RAM | 2 GB (sin Ollama) / 4 GB (con Ollama) | 8 GB+ |
| Disco | 20 GB | 40 GB+ |
| Puertos | 22, 80, 443 | 22, 80, 443 |

Los dominios deben apuntar a la IP del servidor **antes** de ejecutar el instalador (necesario para la emisión de certificados TLS).

---

### Opción 1 — Una sola línea (recomendado)

Descarga y ejecuta el instalador directamente desde GitHub. Se instala en `/opt/cloudcontrol` por defecto.

```bash
curl -fsSL https://raw.githubusercontent.com/henrichile/CloudControl-vertex/main/install.sh | sudo bash
```

El instalador hará las siguientes preguntas de forma interactiva:

```
? Dominio principal para Cloud Control (ej: panel.tudominio.com):
? Correo para Let's Encrypt:
? Dominio del dashboard Traefik [traefik.tudominio.com]:
? Contraseña para el dashboard de Traefik (usuario: admin):
? ¿Cómo deseas configurar Ollama?
    1) Instalar Ollama en este servidor
    2) Usar servidor Ollama externo
    3) Omitir
```

---

### Opción 2 — Descarga y revisión previa

```bash
# Descargar el script
curl -fsSL https://raw.githubusercontent.com/henrichile/CloudControl-vertex/main/install.sh -o install.sh

# Revisar el contenido antes de ejecutar
cat install.sh

# Ejecutar
sudo bash install.sh
```

---

### Opción 3 — Desatendido / CI / Ansible

Todas las variables de configuración se pueden pasar como variables de entorno, evitando las preguntas interactivas. Útil para pipelines de CI/CD o playbooks de Ansible.

```bash
CC_DOMAIN=panel.cloudcontrol.cl \
ACME_EMAIL=alertas@cloudcontrol.cl \
TRAEFIK_DOMAIN=traefik.cloudcontrol.cl \
TRAEFIK_ADMIN_PASS=MiPasswordSegura123 \
OLLAMA_MODE=1 \
OLLAMA_MODEL_CHOICE=1 \
sudo -E bash install.sh
```

#### Variables de configuración

| Variable | Default | Descripción |
|---|---|---|
| `CC_DOMAIN` | — | Dominio principal del panel (obligatorio) |
| `ACME_EMAIL` | — | Correo para notificaciones Let's Encrypt (obligatorio) |
| `TRAEFIK_DOMAIN` | `traefik.<CC_DOMAIN>` | Dominio del dashboard de Traefik |
| `TRAEFIK_ADMIN_PASS` | — | Contraseña Basic Auth del dashboard (obligatorio) |
| `OLLAMA_MODE` | — | `1` local, `2` externo, `3` omitir |
| `OLLAMA_MODEL_CHOICE` | `1` | Índice del modelo (ver tabla) |
| `OLLAMA_MODEL` | `llama3` | Nombre exacto del modelo (anula `OLLAMA_MODEL_CHOICE`) |
| `OLLAMA_HOST_CUSTOM` | — | URL del servidor Ollama externo (solo con `OLLAMA_MODE=2`) |
| `CC_INSTALL_DIR` | `/opt/cloudcontrol` | Directorio de instalación |
| `CC_BRANCH` | `main` | Rama de GitHub a usar |

#### Modelos Ollama disponibles

| `OLLAMA_MODEL_CHOICE` | Modelo | Tamaño | Descripción |
|---|---|---|---|
| `1` | `llama3` | ~4.7 GB | Uso general — **recomendado** |
| `2` | `mistral` | ~4.1 GB | Rápido, buena calidad |
| `3` | `llama3:8b` | ~4.7 GB | Llama 3, 8B parámetros |
| `4` | `gemma2:9b` | ~5.4 GB | Google Gemma 2 |
| `5` | `qwen2:7b` | ~4.4 GB | Alibaba Qwen2 |
| `6` | (nombre manual) | — | Cualquier modelo disponible en Ollama |

---

### Opción 4 — Rama o directorio personalizado

```bash
# Instalar desde una rama específica
CC_BRANCH=develop \
curl -fsSL https://raw.githubusercontent.com/henrichile/CloudControl-vertex/develop/install.sh | sudo bash

# Instalar en un directorio personalizado
CC_INSTALL_DIR=/srv/cloudcontrol \
curl -fsSL https://raw.githubusercontent.com/henrichile/CloudControl-vertex/main/install.sh | sudo bash
```

---

### Lo que hace el instalador

```
▶ Detectando sistema operativo
▶ Configuración de Cloud Control        ← preguntas interactivas (o variables de entorno)
▶ Configuración de AIOps (Ollama)
▶ Instalando dependencias del sistema   ← apt / dnf según el OS
▶ Obteniendo Cloud Control desde GitHub ← git clone / git pull
▶ Verificando Docker                    ← instala Docker CE si no está presente
▶ Instalando Ollama                     ← solo si se eligió modo local
▶ Descargando modelo <nombre>           ← ollama pull
▶ Configurando red Docker compartida    ← crea la red traefik-public
▶ Configurando firewall                 ← UFW (Ubuntu/Debian) o firewalld (AlmaLinux/RHEL)
▶ Generando secretos                    ← JWT_SECRET + hash bcrypt para Traefik
▶ Escribiendo archivos de configuración ← .env, infrastructure/.env, override.yml
▶ Levantando Traefik v3
▶ Construyendo Cloud Control
▶ Esperando que el backend esté saludable
▶ Verificación de servicios
```

---

### Sistemas operativos soportados

| OS | Versión | Paquetes | Firewall |
|---|---|---|---|
| Ubuntu | 22.04, 24.04 | apt | UFW |
| Debian | 11, 12 | apt | UFW |
| AlmaLinux | 8, 9, 10 | dnf | firewalld |
| Rocky Linux | 8, 9 | dnf | firewalld |
| RHEL | 8, 9 | dnf | firewalld |

---

### Actualizar a una nueva versión

```bash
cd /opt/cloudcontrol
git pull
docker compose up -d --build
```

---

## Comandos útiles post-instalación

```bash
cd /opt/cloudcontrol

# Logs de la aplicación
docker compose logs -f

# Logs del backend únicamente
docker compose logs -f backend

# Reiniciar servicios
docker compose restart

# Ver estado de contenedores
docker compose ps

# Logs de Traefik
docker logs -f traefik

# Logs de Ollama
journalctl -u ollama -f

# Descargar un modelo adicional
ollama pull codellama

# Ver modelos instalados
ollama list
```

---

## Stacks de proyectos disponibles

Al crear un proyecto desde el panel o la CLI, Cloud Control genera automáticamente el `docker-compose.yml` y los archivos de configuración necesarios.

| Stack | Servicios | Traefik | Archivos adicionales |
|---|---|---|---|
| `MERN` | MongoDB + Express + React + Node | ✔ | — |
| `LAMP` | Apache + MySQL + PHP 8 | ✔ | — |
| `laravel-redis` | Laravel + MySQL + Redis + Nginx | ✔ | — |
| `fastapi-postgres` | FastAPI + PostgreSQL | ✔ | — |
| `data-science` | Jupyter Lab + PostgreSQL + pgAdmin | ✔ | — |
| `nextjs-prisma` | Next.js + Prisma + PostgreSQL | ✔ | — |
| `django-postgres` | Django + PostgreSQL + Redis + Celery | ✔ | — |
| `spring-boot` | Spring Boot + PostgreSQL + pgAdmin | ✔ | — |
| `rails-postgres` | Rails + PostgreSQL + Redis + Sidekiq | ✔ | — |
| `wordpress` | FrankenPHP + MySQL 8 + Redis + phpMyAdmin | ✔ | Caddyfile, opcache.ini, php.ini, my.cnf |
| `rust-actix` | Actix-Web + PostgreSQL + Redis | ✔ | — |
| `go-postgres` | Go (Gin/Chi) + PostgreSQL + Redis | ✔ | — |
| `elk` | Elasticsearch + Logstash + Kibana | ✔ | — |

Cuando el proyecto se crea con un dominio, Traefik inyecta automáticamente las labels de enrutamiento y TLS.

---

## CLI — cloudctl

```bash
# Autenticación
cloudctl login

# Contenedores
cloudctl containers list
cloudctl containers start <nombre>
cloudctl containers stop <nombre>
cloudctl containers logs <nombre> --follow

# Proyectos
cloudctl projects list
cloudctl projects create --name mi-app --stack MERN --domain app.ejemplo.com
cloudctl projects up   --name mi-app
cloudctl projects down --name mi-app

# AIOps
cloudctl aiops analyze <contenedor>
cloudctl aiops audit --file docker-compose.yml
cloudctl aiops logs  <contenedor>
```

---

## API REST

```
GET    /api/v1/health

# Contenedores
GET    /api/v1/containers
GET    /api/v1/containers/:id
POST   /api/v1/containers/:id/start
POST   /api/v1/containers/:id/stop
DELETE /api/v1/containers/:id
GET    /api/v1/containers/:id/logs
GET    /api/v1/containers/:id/stats     (WebSocket)

# Proyectos
GET    /api/v1/stacks
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/:id
POST   /api/v1/projects/:id/up
POST   /api/v1/projects/:id/down
DELETE /api/v1/projects/:id

# AIOps
POST   /api/v1/aiops/analyze
POST   /api/v1/aiops/audit
POST   /api/v1/aiops/logs
```

---

## Desarrollo local

```bash
# Prerrequisitos: Go 1.22+, Node.js 20+, Docker, Ollama

git clone https://github.com/henrichile/CloudControl-vertex
cd CloudControl-vertex

# Instalar dependencias y verificar compilación
make install

# Levantar entorno de desarrollo completo (con hot reload)
make docker-dev

# O por separado:
make dev-backend    # backend con Air (hot reload)
make dev-frontend   # frontend con Vite

# Tests
make test

# Build de producción
make build
```

### Variables de entorno para desarrollo

```bash
# backend/.env
PORT=8080
DATABASE_PATH=./cloudcontrol.db
DOCKER_HOST=unix:///var/run/docker.sock
OLLAMA_HOST=http://localhost:11434
OLLAMA_MODEL=llama3
JWT_SECRET=dev-secret-cambiar-en-produccion
```

---

## Licencia

MIT © Etasoft / henrichile
