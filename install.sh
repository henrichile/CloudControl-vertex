#!/usr/bin/env bash
# =============================================================================
#  Cloud Control — Instalador de producción
#  Soporta: Ubuntu 22.04 / 24.04, Debian 11 / 12
# =============================================================================
set -euo pipefail
IFS=$'\n\t'

# ── Colores y utilidades de output ────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

info()   { echo -e "${GREEN}  ✔${NC}  $*"; }
warn()   { echo -e "${YELLOW}  ⚠${NC}  $*"; }
error()  { echo -e "${RED}  ✖${NC}  $*" >&2; exit 1; }
step()   { echo -e "\n${BOLD}${BLUE}▶ $*${NC}"; }
prompt() { echo -e "${CYAN}  ?${NC}  $*"; }

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$REPO_DIR/infrastructure"
ACME_FILE="$INFRA_DIR/acme.json"

# ── Banner ────────────────────────────────────────────────────────────────────
echo -e "${BOLD}"
cat <<'BANNER'
   _____ _                 _  _____            _             _
  / ____| |               | |/ ____|          | |           | |
 | |    | | ___  _   _  __| | |     ___  _ __ | |_ _ __ ___ | |
 | |    | |/ _ \| | | |/ _` | |    / _ \| '_ \| __| '__/ _ \| |
 | |____| | (_) | |_| | (_| | |___| (_) | | | | |_| | | (_) | |
  \_____|_|\___/ \__,_|\__,_|\_____\___/|_| |_|\__|_|  \___/|_|

BANNER
echo -e "${NC}${BOLD}  Instalador de producción — Cloud Control${NC}"
echo -e "  $(date '+%Y-%m-%d %H:%M:%S')\n"

# ── Verificar root ────────────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && error "Ejecutar como root: sudo bash install.sh"

# ── Detectar OS ───────────────────────────────────────────────────────────────
step "Detectando sistema operativo"
[[ -f /etc/os-release ]] || error "/etc/os-release no encontrado"
# shellcheck source=/dev/null
source /etc/os-release
OS_ID="${ID:-unknown}"
info "Detectado: $PRETTY_NAME"

case "$OS_ID" in
    ubuntu|debian) ;;
    *) warn "OS no verificado ($OS_ID). Continuando…" ;;
esac

# ── Verificar RAM ─────────────────────────────────────────────────────────────
RAM_MB=$(awk '/MemTotal/ {printf "%d", $2/1024}' /proc/meminfo)
if [[ $RAM_MB -lt 3800 ]]; then
    warn "RAM disponible: ${RAM_MB} MB — Ollama local requiere al menos 4 GB"
else
    info "RAM: ${RAM_MB} MB"
fi

# ── Recopilar configuración ───────────────────────────────────────────────────
step "Configuración de Cloud Control"

# Dominio principal
if [[ -z "${CC_DOMAIN:-}" ]]; then
    prompt "Dominio principal para Cloud Control (ej: panel.tudominio.com):"
    read -r CC_DOMAIN
fi
[[ -z "$CC_DOMAIN" ]] && error "El dominio no puede estar vacío"

# Correo Let's Encrypt
if [[ -z "${ACME_EMAIL:-}" ]]; then
    prompt "Correo para Let's Encrypt:"
    read -r ACME_EMAIL
fi
[[ -z "$ACME_EMAIL" ]] && error "El correo no puede estar vacío"

# Dominio dashboard Traefik
if [[ -z "${TRAEFIK_DOMAIN:-}" ]]; then
    _default_traefik="traefik.$CC_DOMAIN"
    prompt "Dominio del dashboard Traefik [${_default_traefik}]:"
    read -r TRAEFIK_DOMAIN
    TRAEFIK_DOMAIN="${TRAEFIK_DOMAIN:-$_default_traefik}"
fi

# Contraseña Traefik dashboard
if [[ -z "${TRAEFIK_ADMIN_PASS:-}" ]]; then
    prompt "Contraseña para el dashboard de Traefik (usuario: admin):"
    read -rs TRAEFIK_ADMIN_PASS
    echo
fi
[[ -z "$TRAEFIK_ADMIN_PASS" ]] && error "La contraseña no puede estar vacía"

# ── Configuración Ollama ───────────────────────────────────────────────────────
step "Configuración de AIOps (Ollama)"

INSTALL_OLLAMA_LOCAL=false
OLLAMA_HOST="http://host.docker.internal:11434"
OLLAMA_MODEL="${OLLAMA_MODEL:-llama3}"

if [[ -z "${OLLAMA_MODE:-}" ]]; then
    echo
    echo -e "  ${BOLD}¿Cómo deseas configurar Ollama (IA local)?${NC}"
    echo -e "  ${CYAN}1)${NC} Instalar Ollama en este servidor (recomendado, requiere ≥4 GB RAM)"
    echo -e "  ${CYAN}2)${NC} Usar un servidor Ollama externo (URL personalizada)"
    echo -e "  ${CYAN}3)${NC} Omitir — no usar AIOps por ahora"
    echo
    prompt "Elige una opción [1/2/3]:"
    read -r OLLAMA_MODE
fi

case "${OLLAMA_MODE:-1}" in
    1)
        INSTALL_OLLAMA_LOCAL=true
        # Selección de modelo
        if [[ -z "${OLLAMA_MODEL_CHOICE:-}" ]]; then
            echo
            echo -e "  ${BOLD}Modelo a descargar:${NC}"
            echo -e "  ${CYAN}1)${NC} llama3        (~4.7 GB) — uso general, recomendado"
            echo -e "  ${CYAN}2)${NC} mistral       (~4.1 GB) — rápido, buena calidad"
            echo -e "  ${CYAN}3)${NC} llama3:8b     (~4.7 GB) — misma familia, 8B parámetros"
            echo -e "  ${CYAN}4)${NC} gemma2:9b     (~5.4 GB) — Google Gemma 2"
            echo -e "  ${CYAN}5)${NC} qwen2:7b      (~4.4 GB) — Alibaba Qwen2"
            echo -e "  ${CYAN}6)${NC} Otro (ingresar nombre manualmente)"
            echo
            prompt "Elige modelo [1]:"
            read -r OLLAMA_MODEL_CHOICE
        fi
        case "${OLLAMA_MODEL_CHOICE:-1}" in
            1) OLLAMA_MODEL="llama3" ;;
            2) OLLAMA_MODEL="mistral" ;;
            3) OLLAMA_MODEL="llama3:8b" ;;
            4) OLLAMA_MODEL="gemma2:9b" ;;
            5) OLLAMA_MODEL="qwen2:7b" ;;
            6)
                prompt "Nombre del modelo (ej: phi3, codellama):"
                read -r OLLAMA_MODEL
                [[ -z "$OLLAMA_MODEL" ]] && OLLAMA_MODEL="llama3"
                ;;
            *) OLLAMA_MODEL="llama3" ;;
        esac
        OLLAMA_HOST="http://host.docker.internal:11434"
        info "Ollama local — modelo: $OLLAMA_MODEL"
        ;;
    2)
        if [[ -z "${OLLAMA_HOST_CUSTOM:-}" ]]; then
            prompt "URL del servidor Ollama externo [http://localhost:11434]:"
            read -r OLLAMA_HOST_CUSTOM
        fi
        OLLAMA_HOST="${OLLAMA_HOST_CUSTOM:-http://localhost:11434}"
        if [[ -z "${OLLAMA_MODEL_CUSTOM:-}" ]]; then
            prompt "Nombre del modelo a usar [llama3]:"
            read -r OLLAMA_MODEL_CUSTOM
        fi
        OLLAMA_MODEL="${OLLAMA_MODEL_CUSTOM:-llama3}"
        info "Ollama externo — host: $OLLAMA_HOST — modelo: $OLLAMA_MODEL"
        ;;
    3)
        OLLAMA_HOST="http://host.docker.internal:11434"
        warn "AIOps deshabilitado — puedes configurarlo más tarde editando .env"
        ;;
    *)
        warn "Opción no reconocida, omitiendo Ollama"
        ;;
esac

# ── Confirmación ──────────────────────────────────────────────────────────────
echo
info "Dominio Cloud Control  : $CC_DOMAIN"
info "Dominio Traefik        : $TRAEFIK_DOMAIN"
info "Email ACME             : $ACME_EMAIL"
info "Ollama local           : $INSTALL_OLLAMA_LOCAL"
info "Ollama host            : $OLLAMA_HOST"
info "Modelo IA              : $OLLAMA_MODEL"
echo
prompt "¿Continuar con la instalación? [S/n]:"
read -r CONFIRM
[[ "${CONFIRM,,}" == "n" ]] && { echo "Instalación cancelada."; exit 0; }

# ── Instalar paquetes del sistema ─────────────────────────────────────────────
step "Instalando dependencias del sistema"

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
    curl wget git ca-certificates gnupg lsb-release \
    openssl apache2-utils ufw pciutils 2>/dev/null || true

info "Paquetes base instalados"

# ── Instalar Docker CE ────────────────────────────────────────────────────────
step "Verificando Docker"

if command -v docker &>/dev/null; then
    DOCKER_VER=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "desconocida")
    info "Docker ya instalado (versión $DOCKER_VER)"
else
    info "Instalando Docker CE desde repositorio oficial…"
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
        https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
        | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io \
        docker-buildx-plugin docker-compose-plugin
    systemctl enable --now docker
    info "Docker CE instalado y activo"
fi

docker compose version &>/dev/null || error "Docker Compose v2 no disponible"
info "Docker Compose v2 disponible"

# ── Instalar Ollama local ─────────────────────────────────────────────────────
if [[ "$INSTALL_OLLAMA_LOCAL" == "true" ]]; then
    step "Instalando Ollama"

    if command -v ollama &>/dev/null; then
        OLLAMA_VER=$(ollama --version 2>/dev/null || echo "desconocida")
        info "Ollama ya instalado ($OLLAMA_VER)"
    else
        info "Descargando e instalando Ollama…"
        curl -fsSL https://ollama.com/install.sh | sh
        info "Ollama instalado"
    fi

    # Detectar GPU (informativo)
    if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
        GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
        info "GPU NVIDIA detectada: ${GPU_NAME} — Ollama usará aceleración CUDA"
    elif lspci 2>/dev/null | grep -qi "amd.*radeon\|radeon.*amd"; then
        info "GPU AMD detectada — Ollama usará aceleración ROCm si está disponible"
    else
        warn "No se detectó GPU dedicada — Ollama correrá en CPU (más lento)"
    fi

    # Habilitar y arrancar el servicio systemd
    systemctl enable ollama 2>/dev/null || true
    systemctl start  ollama 2>/dev/null || true

    # Esperar a que la API de Ollama esté lista
    info "Esperando que Ollama esté disponible en :11434…"
    for i in {1..24}; do
        if curl -sf http://localhost:11434/api/tags &>/dev/null; then
            info "Ollama responde en http://localhost:11434"
            break
        fi
        [[ $i -eq 24 ]] && error "Ollama no respondió tras 120 s — revisa: journalctl -u ollama"
        sleep 5
    done

    # Descargar el modelo elegido
    step "Descargando modelo $OLLAMA_MODEL (puede tardar varios minutos)"
    if ollama list 2>/dev/null | grep -q "^${OLLAMA_MODEL}"; then
        info "Modelo '$OLLAMA_MODEL' ya descargado"
    else
        ollama pull "$OLLAMA_MODEL"
        info "Modelo '$OLLAMA_MODEL' descargado"
    fi
fi

# ── Red traefik-public ────────────────────────────────────────────────────────
step "Configurando red Docker compartida"

if docker network inspect traefik-public &>/dev/null; then
    info "Red 'traefik-public' ya existe"
else
    docker network create traefik-public
    info "Red 'traefik-public' creada"
fi

# ── Firewall UFW ──────────────────────────────────────────────────────────────
step "Configurando firewall (UFW)"

if command -v ufw &>/dev/null; then
    ufw --force enable                  2>/dev/null || true
    ufw allow 22/tcp    comment "SSH"   2>/dev/null || true
    ufw allow 80/tcp    comment "HTTP"  2>/dev/null || true
    ufw allow 443/tcp   comment "HTTPS" 2>/dev/null || true
    ufw allow 443/udp   comment "HTTP3" 2>/dev/null || true
    # Bloquear acceso externo al puerto de Ollama (solo accesible desde contenedores)
    ufw deny  11434     comment "Ollama — solo acceso interno" 2>/dev/null || true
    info "Reglas UFW: 22, 80, 443 (tcp+udp) | 11434 bloqueado externamente"
else
    warn "UFW no disponible — configura el firewall manualmente"
fi

# ── Generar secretos ──────────────────────────────────────────────────────────
step "Generando secretos"

JWT_SECRET=$(openssl rand -hex 32)
info "JWT_SECRET generado"

TRAEFIK_HASH=$(htpasswd -nbB admin "$TRAEFIK_ADMIN_PASS" | sed -e 's/\$/\$\$/g')
info "Hash Basic Auth generado"

# ── Escribir archivos de configuración ────────────────────────────────────────
step "Escribiendo archivos de configuración"

# .env raíz — Cloud Control
cat > "$REPO_DIR/.env" <<EOF
# Cloud Control — producción
# Generado por install.sh el $(date '+%Y-%m-%d %H:%M:%S')

JWT_SECRET=${JWT_SECRET}
OLLAMA_HOST=${OLLAMA_HOST}
OLLAMA_MODEL=${OLLAMA_MODEL}
CC_DOMAIN=${CC_DOMAIN}
EOF
chmod 600 "$REPO_DIR/.env"
info ".env raíz escrito"

# infrastructure/.env — Traefik
cat > "$INFRA_DIR/.env" <<EOF
# Traefik — producción
# Generado por install.sh el $(date '+%Y-%m-%d %H:%M:%S')

ACME_EMAIL=${ACME_EMAIL}
TRAEFIK_DOMAIN=${TRAEFIK_DOMAIN}
TRAEFIK_DASHBOARD_AUTH=${TRAEFIK_HASH}
EOF
chmod 600 "$INFRA_DIR/.env"
info "infrastructure/.env escrito"

# acme.json — almacén de certificados TLS
[[ ! -f "$ACME_FILE" ]] && touch "$ACME_FILE"
chmod 600 "$ACME_FILE"
info "acme.json listo (permisos 600)"

# docker-compose.override.yml
# - Añade labels Traefik al frontend
# - Añade extra_hosts al backend para que resuelva host.docker.internal en Linux
#   (necesario para que los contenedores alcancen Ollama en el host)
cat > "$REPO_DIR/docker-compose.override.yml" <<EOF
# Generado por install.sh — sobrescritura de producción con Traefik + Ollama
# No editar manualmente; re-ejecuta install.sh para regenerar.

services:
  backend:
    extra_hosts:
      # Permite que el backend resuelva 'host.docker.internal' en Linux,
      # necesario para conectarse a Ollama corriendo en el host.
      - "host.docker.internal:host-gateway"

  frontend:
    ports: []
    networks:
      - internal
      - traefik-public
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.cloudcontrol.rule=Host(\`${CC_DOMAIN}\`)"
      - "traefik.http.routers.cloudcontrol.entrypoints=websecure"
      - "traefik.http.routers.cloudcontrol.tls.certresolver=letsencrypt"
      - "traefik.http.services.cloudcontrol.loadbalancer.server.port=80"
      - "traefik.http.routers.cloudcontrol.middlewares=secure-headers@docker"

networks:
  traefik-public:
    external: true
EOF
info "docker-compose.override.yml escrito"

# ── Levantar Traefik ──────────────────────────────────────────────────────────
step "Levantando Traefik v3"

docker compose \
    --env-file "$INFRA_DIR/.env" \
    -f "$INFRA_DIR/docker-compose.traefik.yml" \
    up -d --pull always

for i in {1..12}; do
    if docker ps --filter "name=traefik" --filter "status=running" | grep -q traefik; then
        info "Traefik en ejecución"
        break
    fi
    [[ $i -eq 12 ]] && error "Traefik no arrancó — revisa: docker logs traefik"
    sleep 5
done

# ── Construir y levantar Cloud Control ────────────────────────────────────────
step "Construyendo Cloud Control (puede tardar varios minutos)"

docker compose \
    --env-file "$REPO_DIR/.env" \
    -f "$REPO_DIR/docker-compose.yml" \
    -f "$REPO_DIR/docker-compose.override.yml" \
    up -d --build

# Esperar healthcheck del backend
step "Esperando que el backend esté saludable"
MAX_WAIT=120
ELAPSED=0
until docker compose \
        --env-file "$REPO_DIR/.env" \
        -f "$REPO_DIR/docker-compose.yml" \
        ps backend 2>/dev/null | grep -q "healthy"; do
    if [[ $ELAPSED -ge $MAX_WAIT ]]; then
        warn "El backend tardó más de ${MAX_WAIT}s — puede que aún esté iniciando"
        break
    fi
    printf "."
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done
echo

# ── Verificación final ────────────────────────────────────────────────────────
step "Verificación de servicios"

check_container() {
    local name="$1"
    if docker ps --filter "name=$name" --filter "status=running" | grep -q "$name"; then
        info "Contenedor '$name' corriendo"
    else
        warn "Contenedor '$name' no encontrado o no está corriendo"
    fi
}

check_container "traefik"
check_container "cloudcontrol-backend"
check_container "cloudcontrol-frontend"

if [[ "$INSTALL_OLLAMA_LOCAL" == "true" ]]; then
    if systemctl is-active --quiet ollama; then
        info "Servicio ollama activo (systemd)"
    else
        warn "Servicio ollama no activo — revisa: journalctl -u ollama"
    fi
fi

# ── Resumen ───────────────────────────────────────────────────────────────────
echo
echo -e "${BOLD}${GREEN}════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}  Cloud Control instalado correctamente${NC}"
echo -e "${BOLD}${GREEN}════════════════════════════════════════════════════════${NC}"
echo
echo -e "  ${BOLD}Panel principal:${NC}    https://${CC_DOMAIN}"
echo -e "  ${BOLD}Dashboard Traefik:${NC}  https://${TRAEFIK_DOMAIN}"
echo -e "                      Usuario: admin | Contraseña: (la que ingresaste)"
if [[ "$INSTALL_OLLAMA_LOCAL" == "true" ]]; then
echo -e "  ${BOLD}Ollama local:${NC}       http://localhost:11434"
echo -e "  ${BOLD}Modelo cargado:${NC}     $OLLAMA_MODEL"
fi
echo
echo -e "  ${BOLD}Archivos de configuración:${NC}"
echo -e "    $REPO_DIR/.env                — Cloud Control"
echo -e "    $INFRA_DIR/.env       — Traefik"
echo -e "    $INFRA_DIR/acme.json  — Certificados TLS"
echo
echo -e "  ${BOLD}Comandos útiles:${NC}"
echo -e "    Logs app:       docker compose logs -f"
echo -e "    Reiniciar:      docker compose restart"
echo -e "    Actualizar:     git pull && docker compose up -d --build"
echo -e "    Logs Traefik:   docker logs -f traefik"
if [[ "$INSTALL_OLLAMA_LOCAL" == "true" ]]; then
echo -e "    Logs Ollama:    journalctl -u ollama -f"
echo -e "    Cargar modelo:  ollama pull <nombre>"
echo -e "    Listar modelos: ollama list"
fi
echo
echo -e "  ${YELLOW}Nota:${NC} Los certificados TLS pueden tardar 1-2 minutos la primera vez."
echo
