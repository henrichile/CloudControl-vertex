#!/usr/bin/env bash
# Cloud Control — inicialización de Traefik como reverse proxy global
# Ejecutar una sola vez en el servidor de producción.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_NAME="traefik-public"
ACME_FILE="$SCRIPT_DIR/acme.json"
ENV_FILE="$SCRIPT_DIR/.env"

# ── Colores ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ── Prerequisitos ─────────────────────────────────────────────────────────────
command -v docker >/dev/null 2>&1 || error "Docker no está instalado"
docker compose version >/dev/null 2>&1 || error "Docker Compose v2 no está disponible"

# ── .env ──────────────────────────────────────────────────────────────────────
if [[ ! -f "$ENV_FILE" ]]; then
    warn ".env no encontrado — generando plantilla en $ENV_FILE"
    cat > "$ENV_FILE" <<'EOF'
# Correo para Let's Encrypt (notificaciones de vencimiento)
ACME_EMAIL=admin@ejemplo.com

# Dominio del dashboard de Traefik
TRAEFIK_DOMAIN=traefik.tudominio.com

# Usuario:hash para Basic Auth del dashboard
# Generar con: echo $(htpasswd -nB admin) | sed -e 's/\$/\$\$/g'
TRAEFIK_DASHBOARD_AUTH=admin:$$2y$$05$$examplehashchangeme
EOF
    info "Edita $ENV_FILE con tus valores y vuelve a ejecutar este script."
    exit 0
fi

# Verificar variables obligatorias
# shellcheck source=/dev/null
source "$ENV_FILE"
[[ -z "${ACME_EMAIL:-}" ]] && error "ACME_EMAIL no está definido en .env"
[[ -z "${TRAEFIK_DASHBOARD_AUTH:-}" ]] && error "TRAEFIK_DASHBOARD_AUTH no está definido en .env"

# ── Red Docker compartida ─────────────────────────────────────────────────────
if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
    info "Red '$NETWORK_NAME' ya existe — omitiendo creación"
else
    info "Creando red Docker '$NETWORK_NAME'..."
    docker network create "$NETWORK_NAME"
fi

# ── acme.json — almacén de certificados TLS ───────────────────────────────────
if [[ ! -f "$ACME_FILE" ]]; then
    info "Creando $ACME_FILE con permisos 600..."
    touch "$ACME_FILE"
fi
chmod 600 "$ACME_FILE"

# ── Levantar Traefik ──────────────────────────────────────────────────────────
info "Levantando Traefik v3..."
docker compose \
    --env-file "$ENV_FILE" \
    -f "$SCRIPT_DIR/docker-compose.traefik.yml" \
    up -d --pull always

# ── Verificación ──────────────────────────────────────────────────────────────
sleep 3
if docker ps --filter "name=traefik" --filter "status=running" | grep -q traefik; then
    info "Traefik está corriendo."
    info "Dashboard: https://${TRAEFIK_DOMAIN:-traefik.localhost}"
    info ""
    info "Ahora puedes crear proyectos con dominio desde Cloud Control y el"
    info "tráfico será enrutado automáticamente mediante labels Docker."
else
    error "Traefik no inició correctamente. Revisa: docker logs traefik"
fi
