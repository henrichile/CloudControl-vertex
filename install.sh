#!/usr/bin/env bash
# =============================================================================
#  Cloud Control — Instalador de producción
#  Repositorio: https://github.com/henrichile/CloudControl-vertex
#  Soporta: Ubuntu 22.04/24.04, Debian 11/12, AlmaLinux 8/9/10, Rocky 8/9
#
#  Uso rápido (una sola línea):
#    curl -fsSL https://raw.githubusercontent.com/henrichile/CloudControl-vertex/main/install.sh | sudo bash
#
#  Uso con variables (desatendido):
#    CC_DOMAIN=panel.ejemplo.com ACME_EMAIL=admin@ejemplo.com \
#    TRAEFIK_ADMIN_PASS=MiPass123 OLLAMA_MODE=1 \
#    sudo -E bash install.sh
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

# ── Entrada de teclado robusta frente a curl | bash ───────────────────────────
# Cuando el script llega por pipe, stdin es el propio contenido del script y
# los `read` consumirían líneas de código en vez del teclado del usuario.
# Abrimos /dev/tty explícitamente como fd 3 para todas las lecturas interactivas.
if [[ -t 0 ]]; then
    exec 3<&0          # stdin ya es un terminal; reusarlo
else
    exec 3</dev/tty    # venimos de pipe (curl | bash); usar el terminal real
fi
tty_read() { read "$@" <&3; }

# ── Constantes del proyecto ───────────────────────────────────────────────────
REPO_URL="https://github.com/henrichile/CloudControl-vertex"
REPO_BRANCH="${CC_BRANCH:-main}"
INSTALL_DIR="${CC_INSTALL_DIR:-/opt/cloudcontrol}"
INFRA_DIR="$INSTALL_DIR/infrastructure"
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
echo -e "  Repositorio: ${CYAN}${REPO_URL}${NC}"
echo -e "  $(date '+%Y-%m-%d %H:%M:%S')\n"

# ── Verificar root ────────────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && error "Ejecutar como root: sudo bash install.sh"

# ── Modo forzado: limpia instalación previa antes de continuar ────────────────
if [[ "${CC_FORCE:-false}" == "true" ]]; then
    step "Modo forzado — limpiando instalación previa"

    if [[ -d "$INSTALL_DIR" ]]; then
        # Bajar contenedores de Cloud Control si existen
        if [[ -f "$INSTALL_DIR/docker-compose.yml" ]]; then
            warn "Deteniendo contenedores de Cloud Control…"
            docker compose \
                -f "$INSTALL_DIR/docker-compose.yml" \
                -f "$INSTALL_DIR/docker-compose.override.yml" \
                down --remove-orphans 2>/dev/null || true
        fi
        warn "Eliminando directorio $INSTALL_DIR…"
        rm -rf "$INSTALL_DIR"
        info "Directorio eliminado"
    else
        info "No existía instalación previa en $INSTALL_DIR"
    fi

    # Bajar Traefik si está corriendo
    if docker ps --filter "name=traefik" --filter "status=running" | grep -q traefik; then
        warn "Deteniendo Traefik…"
        if [[ -f "/opt/traefik/docker-compose.traefik.yml" ]]; then
            docker compose -f /opt/traefik/docker-compose.traefik.yml down 2>/dev/null || true
        else
            docker rm -f traefik 2>/dev/null || true
        fi
        info "Traefik detenido"
    fi

    info "Limpieza completada — iniciando instalación desde cero"
fi

# ── Detectar OS y gestor de paquetes ─────────────────────────────────────────
step "Detectando sistema operativo"
[[ -f /etc/os-release ]] || error "/etc/os-release no encontrado"
# shellcheck source=/dev/null
source /etc/os-release
OS_ID="${ID:-unknown}"
OS_VERSION_ID="${VERSION_ID:-0}"
OS_MAJOR="${OS_VERSION_ID%%.*}"
info "Detectado: $PRETTY_NAME"

if command -v apt-get &>/dev/null; then
    PKG_FAMILY="apt"
elif command -v dnf &>/dev/null; then
    PKG_FAMILY="dnf"
elif command -v yum &>/dev/null; then
    PKG_FAMILY="yum"
else
    error "No se encontró apt, dnf ni yum. OS no soportado."
fi
info "Gestor de paquetes: $PKG_FAMILY"

case "$OS_ID" in
    ubuntu|debian|almalinux|rocky|rhel|centos|fedora) ;;
    *) warn "OS '$OS_ID' no verificado. Continuando…" ;;
esac

# ── Verificar RAM ─────────────────────────────────────────────────────────────
RAM_MB=$(awk '/MemTotal/ {printf "%d", $2/1024}' /proc/meminfo)
[[ $RAM_MB -lt 3800 ]] \
    && warn "RAM: ${RAM_MB} MB — Ollama local requiere al menos 4 GB" \
    || info "RAM: ${RAM_MB} MB"

# ── Funciones de abstracción de paquetes ──────────────────────────────────────
pkg_update() {
    case "$PKG_FAMILY" in
        apt) apt-get update -qq ;;
        dnf) dnf makecache -q ;;
        yum) yum makecache -q ;;
    esac
}

pkg_install() {
    case "$PKG_FAMILY" in
        apt) apt-get install -y -qq "$@" ;;
        dnf) dnf install -y -q "$@" ;;
        yum) yum install -y -q "$@" ;;
    esac
}

# ── Recopilar configuración ───────────────────────────────────────────────────
step "Configuración de Cloud Control"

if [[ -z "${CC_DOMAIN:-}" ]]; then
    prompt "Dominio principal para Cloud Control (ej: panel.tudominio.com):"
    tty_read -r CC_DOMAIN
fi
[[ -z "$CC_DOMAIN" ]] && error "El dominio no puede estar vacío"

if [[ -z "${ACME_EMAIL:-}" ]]; then
    prompt "Correo para Let's Encrypt:"
    tty_read -r ACME_EMAIL
fi
[[ -z "$ACME_EMAIL" ]] && error "El correo no puede estar vacío"

if [[ -z "${TRAEFIK_DOMAIN:-}" ]]; then
    _default_traefik="traefik.$CC_DOMAIN"
    prompt "Dominio del dashboard Traefik [${_default_traefik}]:"
    tty_read -r TRAEFIK_DOMAIN
    TRAEFIK_DOMAIN="${TRAEFIK_DOMAIN:-$_default_traefik}"
fi

if [[ -z "${TRAEFIK_ADMIN_PASS:-}" ]]; then
    prompt "Contraseña para el dashboard de Traefik (usuario: admin):"
    tty_read -rs TRAEFIK_ADMIN_PASS
    echo
fi
[[ -z "$TRAEFIK_ADMIN_PASS" ]] && error "La contraseña no puede estar vacía"

# ── Configuración Ollama ──────────────────────────────────────────────────────
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
    tty_read -r OLLAMA_MODE
fi

case "${OLLAMA_MODE:-1}" in
    1)
        INSTALL_OLLAMA_LOCAL=true
        if [[ -z "${OLLAMA_MODEL_CHOICE:-}" ]]; then
            echo
            echo -e "  ${BOLD}Modelo a descargar:${NC}"
            echo -e "  ${CYAN}1)${NC} llama3     (~4.7 GB) — uso general, recomendado"
            echo -e "  ${CYAN}2)${NC} mistral    (~4.1 GB) — rápido, buena calidad"
            echo -e "  ${CYAN}3)${NC} llama3:8b  (~4.7 GB) — 8B parámetros"
            echo -e "  ${CYAN}4)${NC} gemma2:9b  (~5.4 GB) — Google Gemma 2"
            echo -e "  ${CYAN}5)${NC} qwen2:7b   (~4.4 GB) — Alibaba Qwen2"
            echo -e "  ${CYAN}6)${NC} Otro (nombre manual)"
            echo
            prompt "Elige modelo [1]:"
            tty_read -r OLLAMA_MODEL_CHOICE
        fi
        case "${OLLAMA_MODEL_CHOICE:-1}" in
            1) OLLAMA_MODEL="llama3" ;;
            2) OLLAMA_MODEL="mistral" ;;
            3) OLLAMA_MODEL="llama3:8b" ;;
            4) OLLAMA_MODEL="gemma2:9b" ;;
            5) OLLAMA_MODEL="qwen2:7b" ;;
            6)
                prompt "Nombre del modelo (ej: phi3, codellama):"
                tty_read -r OLLAMA_MODEL
                [[ -z "$OLLAMA_MODEL" ]] && OLLAMA_MODEL="llama3"
                ;;
            *) OLLAMA_MODEL="llama3" ;;
        esac
        OLLAMA_HOST="http://host.docker.internal:11434"
        info "Ollama local — modelo: $OLLAMA_MODEL"
        ;;
    2)
        [[ -z "${OLLAMA_HOST_CUSTOM:-}" ]] && { prompt "URL del servidor Ollama [http://localhost:11434]:"; tty_read -r OLLAMA_HOST_CUSTOM; }
        OLLAMA_HOST="${OLLAMA_HOST_CUSTOM:-http://localhost:11434}"
        [[ -z "${OLLAMA_MODEL_CUSTOM:-}" ]] && { prompt "Modelo a usar [llama3]:"; tty_read -r OLLAMA_MODEL_CUSTOM; }
        OLLAMA_MODEL="${OLLAMA_MODEL_CUSTOM:-llama3}"
        info "Ollama externo — host: $OLLAMA_HOST — modelo: $OLLAMA_MODEL"
        ;;
    3)
        warn "AIOps deshabilitado — configura más tarde editando $INSTALL_DIR/.env"
        ;;
esac

# ── Confirmación ──────────────────────────────────────────────────────────────
echo
info "Directorio de instalación : $INSTALL_DIR"
info "Dominio Cloud Control     : $CC_DOMAIN"
info "Dominio Traefik           : $TRAEFIK_DOMAIN"
info "Email ACME                : $ACME_EMAIL"
info "Ollama local              : $INSTALL_OLLAMA_LOCAL"
[[ "$INSTALL_OLLAMA_LOCAL" == "true" ]] && info "Modelo IA                 : $OLLAMA_MODEL"
echo
prompt "¿Continuar con la instalación? [S/n]:"
tty_read -r CONFIRM
[[ "${CONFIRM,,}" == "n" ]] && { echo "Instalación cancelada."; exit 0; }

# ── Instalar paquetes base (incluye git para el clone) ────────────────────────
step "Instalando dependencias del sistema"

export DEBIAN_FRONTEND=noninteractive
pkg_update

case "$PKG_FAMILY" in
    apt)
        pkg_install curl wget git ca-certificates gnupg lsb-release \
                    openssl apache2-utils ufw pciutils
        ;;
    dnf|yum)
        pkg_install curl wget git ca-certificates gnupg2 \
                    openssl httpd-tools pciutils zstd firewalld
        systemctl enable --now firewalld
        ;;
esac
info "Paquetes base instalados"

# ── Clonar / actualizar repositorio ──────────────────────────────────────────
step "Obteniendo Cloud Control desde GitHub"

if [[ -d "$INSTALL_DIR/.git" ]]; then
    info "Repositorio ya presente en $INSTALL_DIR — actualizando…"
    git -C "$INSTALL_DIR" fetch --quiet origin
    git -C "$INSTALL_DIR" checkout --quiet "$REPO_BRANCH"
    git -C "$INSTALL_DIR" reset --hard "origin/$REPO_BRANCH"
    info "Repositorio actualizado a la rama $REPO_BRANCH"
else
    info "Clonando $REPO_URL → $INSTALL_DIR"
    git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$INSTALL_DIR"
    info "Repositorio clonado"
fi

# ── Instalar Docker CE ────────────────────────────────────────────────────────
step "Verificando Docker"

if command -v docker &>/dev/null; then
    DOCKER_VER=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "desconocida")
    info "Docker ya instalado (versión $DOCKER_VER)"
else
    info "Instalando Docker CE desde repositorio oficial…"

    case "$PKG_FAMILY" in
        apt)
            install -m 0755 -d /etc/apt/keyrings
            curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
                | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            chmod a+r /etc/apt/keyrings/docker.gpg
            echo \
                "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
                https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
                | tee /etc/apt/sources.list.d/docker.list > /dev/null
            apt-get update -qq
            pkg_install docker-ce docker-ce-cli containerd.io \
                        docker-buildx-plugin docker-compose-plugin
            ;;
        dnf|yum)
            pkg_install dnf-plugins-core 2>/dev/null || true
            case "$OS_ID" in
                almalinux|rocky|rhel) _docker_os="rhel" ;;
                fedora)               _docker_os="fedora" ;;
                *)                    _docker_os="centos" ;;
            esac
            dnf config-manager \
                --add-repo "https://download.docker.com/linux/${_docker_os}/docker-ce.repo" \
                2>/dev/null || \
            dnf config-manager \
                --add-repo "https://download.docker.com/linux/centos/docker-ce.repo"
            pkg_install docker-ce docker-ce-cli containerd.io \
                        docker-buildx-plugin docker-compose-plugin
            ;;
    esac

    systemctl enable --now docker
    info "Docker CE instalado y activo"
fi

docker compose version &>/dev/null || error "Docker Compose v2 no disponible"
info "Docker Compose v2 disponible"

# ── Instalar Ollama local ─────────────────────────────────────────────────────
if [[ "$INSTALL_OLLAMA_LOCAL" == "true" ]]; then
    step "Instalando Ollama"

    if command -v ollama &>/dev/null; then
        info "Ollama ya instalado ($(ollama --version 2>/dev/null || echo 'versión desconocida'))"
    else
        info "Descargando e instalando Ollama…"
        curl -fsSL https://ollama.com/install.sh | sh
        info "Ollama instalado"
    fi

    # Detección de GPU (informativo)
    if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null 2>&1; then
        GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
        info "GPU NVIDIA detectada: ${GPU_NAME} — Ollama usará CUDA"
    elif lspci 2>/dev/null | grep -qi "amd.*radeon\|radeon.*amd"; then
        info "GPU AMD detectada — Ollama usará ROCm si está disponible"
    else
        warn "Sin GPU dedicada — Ollama correrá en CPU (más lento)"
    fi

    systemctl enable ollama 2>/dev/null || true
    systemctl start  ollama 2>/dev/null || true

    info "Esperando que Ollama esté disponible en :11434…"
    for i in {1..24}; do
        curl -sf http://localhost:11434/api/tags &>/dev/null && break
        [[ $i -eq 24 ]] && error "Ollama no respondió tras 120 s — revisa: journalctl -u ollama"
        sleep 5
    done
    info "Ollama responde en http://localhost:11434"

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

docker network inspect traefik-public &>/dev/null \
    && info "Red 'traefik-public' ya existe" \
    || { docker network create traefik-public; info "Red 'traefik-public' creada"; }

# ── Firewall ──────────────────────────────────────────────────────────────────
step "Configurando firewall"

if command -v ufw &>/dev/null && [[ "$PKG_FAMILY" == "apt" ]]; then
    ufw --force enable                  2>/dev/null || true
    ufw allow 22/tcp    comment "SSH"   2>/dev/null || true
    ufw allow 80/tcp    comment "HTTP"  2>/dev/null || true
    ufw allow 443/tcp   comment "HTTPS" 2>/dev/null || true
    ufw allow 443/udp   comment "HTTP3" 2>/dev/null || true
    ufw deny  11434     comment "Ollama — solo interno" 2>/dev/null || true
    info "UFW: 22, 80, 443 (tcp+udp) abiertos | 11434 bloqueado"

elif command -v firewall-cmd &>/dev/null \
        || [[ -x /usr/bin/firewall-cmd ]] \
        || [[ -x /usr/sbin/firewall-cmd ]]; then
    FW_CMD=$(command -v firewall-cmd 2>/dev/null \
             || { [[ -x /usr/bin/firewall-cmd ]]  && echo /usr/bin/firewall-cmd; } \
             || echo /usr/sbin/firewall-cmd)
    systemctl enable --now firewalld 2>/dev/null || true
    "$FW_CMD" --permanent --add-service=ssh   2>/dev/null || true
    "$FW_CMD" --permanent --add-service=http  2>/dev/null || true
    "$FW_CMD" --permanent --add-service=https 2>/dev/null || true
    "$FW_CMD" --permanent --add-port=443/udp  2>/dev/null || true
    "$FW_CMD" --permanent --remove-port=11434/tcp 2>/dev/null || true
    "$FW_CMD" --reload 2>/dev/null || true
    info "firewalld: ssh, http, https (tcp+udp) abiertos | 11434 no expuesto"
else
    warn "No se encontró UFW ni firewalld — configura el firewall manualmente"
fi

# ── Generar secretos ──────────────────────────────────────────────────────────
step "Generando secretos"

JWT_SECRET=$(openssl rand -hex 32)
info "JWT_SECRET generado"

TRAEFIK_HASH=$(htpasswd -nbB admin "$TRAEFIK_ADMIN_PASS" | sed -e 's/\$/\$\$/g')
info "Hash Basic Auth generado"

# ── Escribir archivos de configuración ────────────────────────────────────────
step "Escribiendo archivos de configuración"

cat > "$INSTALL_DIR/.env" <<EOF
# Cloud Control — producción
# Generado por install.sh el $(date '+%Y-%m-%d %H:%M:%S')

JWT_SECRET=${JWT_SECRET}
OLLAMA_HOST=${OLLAMA_HOST}
OLLAMA_MODEL=${OLLAMA_MODEL}
CC_DOMAIN=${CC_DOMAIN}
EOF
chmod 600 "$INSTALL_DIR/.env"
info ".env escrito en $INSTALL_DIR"

cat > "$INFRA_DIR/.env" <<EOF
# Traefik — producción
# Generado por install.sh el $(date '+%Y-%m-%d %H:%M:%S')

ACME_EMAIL=${ACME_EMAIL}
TRAEFIK_DOMAIN=${TRAEFIK_DOMAIN}
TRAEFIK_DASHBOARD_AUTH=${TRAEFIK_HASH}
EOF
chmod 600 "$INFRA_DIR/.env"
info "infrastructure/.env escrito"

[[ ! -f "$ACME_FILE" ]] && touch "$ACME_FILE"
chmod 600 "$ACME_FILE"
info "acme.json listo (permisos 600)"

# docker-compose.override.yml:
#   - extra_hosts en backend → los contenedores alcanzan Ollama en el host
#   - labels Traefik + red traefik-public en el frontend
cat > "$INSTALL_DIR/docker-compose.override.yml" <<EOF
# Generado por install.sh — sobrescritura de producción con Traefik + Ollama
# No editar manualmente; re-ejecuta install.sh para regenerar.

services:
  backend:
    extra_hosts:
      # Resuelve host.docker.internal → IP del host en Linux (necesario para Ollama)
      - "host.docker.internal:host-gateway"

  frontend:
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
    docker ps --filter "name=traefik" --filter "status=running" | grep -q traefik \
        && { info "Traefik en ejecución"; break; } \
        || true
    [[ $i -eq 12 ]] && error "Traefik no arrancó — revisa: docker logs traefik"
    sleep 5
done

# ── Construir y levantar Cloud Control ────────────────────────────────────────
step "Construyendo Cloud Control (puede tardar varios minutos)"

docker compose \
    --env-file "$INSTALL_DIR/.env" \
    -f "$INSTALL_DIR/docker-compose.yml" \
    -f "$INSTALL_DIR/docker-compose.override.yml" \
    up -d --build

step "Esperando que el backend esté saludable (hasta 3 min)"
MAX_WAIT=180; ELAPSED=0; BACKEND_OK=false
until docker compose \
        --env-file "$INSTALL_DIR/.env" \
        -f "$INSTALL_DIR/docker-compose.yml" \
        ps backend 2>/dev/null | grep -q "(healthy)"; do

    # Detectar si el contenedor ya salió/crasheó (no vale la pena seguir esperando)
    STATUS=$(docker inspect --format '{{.State.Status}}' cloudcontrol-backend-1 2>/dev/null || echo "missing")
    if [[ "$STATUS" == "exited" || "$STATUS" == "dead" ]]; then
        echo
        error_no_exit() { echo -e "${RED}  ✖${NC}  $*" >&2; }
        error_no_exit "El backend salió inesperadamente. Logs:"
        echo -e "${YELLOW}──────────────────────────────────────────────${NC}"
        docker logs --tail 50 cloudcontrol-backend-1 2>&1 || true
        echo -e "${YELLOW}──────────────────────────────────────────────${NC}"
        error "Instalación fallida. Corrige el error anterior y ejecuta: CC_FORCE=true ... install.sh"
    fi

    if [[ $ELAPSED -ge $MAX_WAIT ]]; then
        echo
        warn "Backend tardó más de ${MAX_WAIT}s. Logs actuales:"
        echo -e "${YELLOW}──────────────────────────────────────────────${NC}"
        docker logs --tail 30 cloudcontrol-backend-1 2>&1 || true
        echo -e "${YELLOW}──────────────────────────────────────────────${NC}"
        warn "Continuando de todas formas — puede que aún esté iniciando"
        break
    fi

    printf "."; sleep 5; ELAPSED=$((ELAPSED+5))
done
echo
[[ $ELAPSED -lt $MAX_WAIT ]] && BACKEND_OK=true

# ── Verificación final ────────────────────────────────────────────────────────
step "Verificación de servicios"

check_container() {
    docker ps --filter "name=$1" --filter "status=running" | grep -q "$1" \
        && info "Contenedor '$1' corriendo" \
        || warn "Contenedor '$1' no encontrado o no está corriendo"
}

check_container "traefik"
check_container "cloudcontrol-backend"
check_container "cloudcontrol-frontend"

if [[ "$INSTALL_OLLAMA_LOCAL" == "true" ]]; then
    systemctl is-active --quiet ollama \
        && info "Servicio ollama activo (systemd)" \
        || warn "Servicio ollama no activo — revisa: journalctl -u ollama"
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
echo -e "  ${BOLD}Sistema operativo:${NC}  $PRETTY_NAME"
echo -e "  ${BOLD}Directorio:${NC}         $INSTALL_DIR"
echo -e "  ${BOLD}Rama:${NC}               $REPO_BRANCH"
echo
echo -e "  ${BOLD}Archivos de configuración:${NC}"
echo -e "    $INSTALL_DIR/.env                — Cloud Control"
echo -e "    $INFRA_DIR/.env          — Traefik"
echo -e "    $INFRA_DIR/acme.json     — Certificados TLS"
echo
echo -e "  ${BOLD}Comandos útiles:${NC}"
echo -e "    cd $INSTALL_DIR"
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
