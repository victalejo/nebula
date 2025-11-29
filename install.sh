#!/bin/bash
set -e

# Nebula PaaS Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/victalejo/nebula/main/install.sh | bash

NEBULA_VERSION="${NEBULA_VERSION:-latest}"
NEBULA_HOME="${NEBULA_HOME:-/opt/nebula}"
NEBULA_USER="${NEBULA_USER:-nebula}"
NEBULA_DOMAIN="${NEBULA_DOMAIN:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[Nebula]${NC} $1"; }
warn() { echo -e "${YELLOW}[Warning]${NC} $1"; }
error() { echo -e "${RED}[Error]${NC} $1"; exit 1; }

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error "Please run as root or with sudo"
    fi
}

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    else
        error "Cannot detect OS. Please install manually."
    fi
    log "Detected OS: $OS $VERSION"
}

# Install Docker if not present
install_docker() {
    if command -v docker &> /dev/null; then
        log "Docker already installed"
        return
    fi

    log "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
    log "Docker installed successfully"
}

# Install Caddy
install_caddy() {
    if command -v caddy &> /dev/null; then
        log "Caddy already installed"
        return
    fi

    log "Installing Caddy..."
    case $OS in
        ubuntu|debian)
            apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
            curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
            curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
            apt-get update
            apt-get install -y caddy
            ;;
        centos|rhel|fedora)
            dnf install -y 'dnf-command(copr)'
            dnf copr enable -y @caddy/caddy
            dnf install -y caddy
            ;;
        *)
            # Manual install
            curl -o /usr/local/bin/caddy -L "https://caddyserver.com/api/download?os=linux&arch=amd64"
            chmod +x /usr/local/bin/caddy
            ;;
    esac
    log "Caddy installed successfully"
}

# Create nebula user
create_user() {
    if id "$NEBULA_USER" &>/dev/null; then
        log "User $NEBULA_USER already exists"
    else
        log "Creating user $NEBULA_USER..."
        useradd -r -s /bin/false -d "$NEBULA_HOME" "$NEBULA_USER"
    fi
    usermod -aG docker "$NEBULA_USER"
}

# Create directories
create_directories() {
    log "Creating directories..."
    mkdir -p "$NEBULA_HOME"/{bin,data,config,logs}
    mkdir -p "$NEBULA_HOME"/data/{apps,builds,compose,databases}
    chown -R "$NEBULA_USER:$NEBULA_USER" "$NEBULA_HOME"
}

# Download Nebula binaries
download_nebula() {
    log "Downloading Nebula $NEBULA_VERSION..."

    if [ "$NEBULA_VERSION" = "latest" ]; then
        DOWNLOAD_URL="https://github.com/victalejo/nebula/releases/latest/download"
    else
        DOWNLOAD_URL="https://github.com/victalejo/nebula/releases/download/$NEBULA_VERSION"
    fi

    # Download server
    curl -fsSL "$DOWNLOAD_URL/nebula-server-linux-amd64" -o "$NEBULA_HOME/bin/nebula-server"
    chmod +x "$NEBULA_HOME/bin/nebula-server"

    # Download CLI
    curl -fsSL "$DOWNLOAD_URL/nebula-linux-amd64" -o "$NEBULA_HOME/bin/nebula"
    chmod +x "$NEBULA_HOME/bin/nebula"

    # Symlink CLI to /usr/local/bin
    ln -sf "$NEBULA_HOME/bin/nebula" /usr/local/bin/nebula

    log "Nebula downloaded successfully"
}

# Generate configuration
generate_config() {
    log "Generating configuration..."

    # Generate random admin password
    ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
    JWT_SECRET=$(openssl rand -base64 32)

    cat > "$NEBULA_HOME/config/config.yaml" << EOF
# Nebula Configuration
server:
  host: 0.0.0.0
  port: 8080

database:
  path: $NEBULA_HOME/data/nebula.db

docker:
  host: unix:///var/run/docker.sock

caddy:
  admin_url: http://localhost:2019

data:
  dir: $NEBULA_HOME/data

auth:
  jwt_secret: $JWT_SECRET
  admin_username: admin
  admin_password: $ADMIN_PASSWORD

logging:
  level: info
  format: json
EOF

    chown "$NEBULA_USER:$NEBULA_USER" "$NEBULA_HOME/config/config.yaml"
    chmod 600 "$NEBULA_HOME/config/config.yaml"

    log "Configuration generated"
    echo ""
    echo "=========================================="
    echo "  Admin Credentials (SAVE THESE!)"
    echo "=========================================="
    echo "  Username: admin"
    echo "  Password: $ADMIN_PASSWORD"
    echo "=========================================="
    echo ""
}

# Configure Caddy
configure_caddy() {
    log "Configuring Caddy..."

    cat > /etc/caddy/Caddyfile << 'EOF'
{
    admin localhost:2019
    auto_https disable_redirects
}

# Default catch-all (will be managed by Nebula API)
:80 {
    respond "Nebula PaaS" 200
}
EOF

    systemctl enable caddy
    systemctl restart caddy
    log "Caddy configured"
}

# Create systemd service
create_systemd_service() {
    log "Creating systemd service..."

    cat > /etc/systemd/system/nebula.service << EOF
[Unit]
Description=Nebula PaaS Server
After=network.target docker.service caddy.service
Requires=docker.service

[Service]
Type=simple
User=$NEBULA_USER
Group=$NEBULA_USER
WorkingDirectory=$NEBULA_HOME
ExecStart=$NEBULA_HOME/bin/nebula-server --config $NEBULA_HOME/config/config.yaml
Restart=always
RestartSec=5
StandardOutput=append:$NEBULA_HOME/logs/nebula.log
StandardError=append:$NEBULA_HOME/logs/nebula.log

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$NEBULA_HOME/data $NEBULA_HOME/logs
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable nebula
    systemctl start nebula

    log "Nebula service created and started"
}

# Configure firewall
configure_firewall() {
    log "Configuring firewall..."

    if command -v ufw &> /dev/null; then
        ufw allow 80/tcp
        ufw allow 443/tcp
        ufw allow 8080/tcp
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --reload
    fi
}

# Main installation
main() {
    echo ""
    echo "  _   _      _           _       "
    echo " | \ | | ___| |__  _   _| | __ _ "
    echo " |  \| |/ _ \ '_ \| | | | |/ _\` |"
    echo " | |\  |  __/ |_) | |_| | | (_| |"
    echo " |_| \_|\___|_.__/ \__,_|_|\__,_|"
    echo ""
    echo " Lightweight PaaS Installer"
    echo ""

    check_root
    detect_os

    log "Starting installation..."

    # Update package manager
    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y curl git
            ;;
        centos|rhel|fedora)
            dnf install -y curl git
            ;;
    esac

    install_docker
    install_caddy
    create_user
    create_directories
    download_nebula
    generate_config
    configure_caddy
    create_systemd_service
    configure_firewall

    # Get server IP
    SERVER_IP=$(hostname -I | awk '{print $1}')

    echo ""
    log "Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Configure your DNS to point to this server ($SERVER_IP)"
    echo "  2. Login with: nebula login http://$SERVER_IP:8080"
    echo "  3. Create your first app: nebula apps create myapp --mode docker_image"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status nebula    # Check status"
    echo "  journalctl -u nebula -f    # View logs"
    echo "  nebula --help              # CLI help"
    echo ""
}

main "$@"
