# Vibeman AI Assistant Container
# Provides development tools and AI-assisted coding environment

FROM ubuntu:22.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install base dependencies and development tools
RUN apt-get update && apt-get install -y \
    # Shell and terminal
    zsh \
    tmux \
    screen \
    # Basic utilities
    curl \
    wget \
    git \
    vim \
    nano \
    tree \
    htop \
    ncdu \
    # Search and find tools
    silversearcher-ag \
    fd-find \
    fzf \
    # JSON/YAML tools
    jq \
    # Build tools
    build-essential \
    python3 \
    python3-pip \
    # Network tools for debugging
    net-tools \
    dnsutils \
    iputils-ping \
    netcat \
    telnet \
    nmap \
    # Process monitoring
    procps \
    psmisc \
    lsof \
    # Log parsing utilities
    less \
    multitail \
    # Text processing
    gawk \
    sed \
    grep \
    # Archive tools
    zip \
    unzip \
    tar \
    # Performance tools
    iotop \
    sysstat \
    # SSH client
    openssh-client \
    && rm -rf /var/lib/apt/lists/*

# Install yq (YAML processor) manually
RUN curl -L "https://github.com/mikefarah/yq/releases/download/v4.43.1/yq_linux_arm64" -o /usr/local/bin/yq \
    && chmod +x /usr/local/bin/yq

# Install exa (modern ls replacement) manually
RUN curl -L "https://github.com/ogham/exa/releases/download/v0.10.1/exa-linux-armv7-v0.10.1.zip" -o exa.zip \
    && unzip -j exa.zip bin/exa -d /usr/local/bin/ \
    && chmod +x /usr/local/bin/exa \
    && rm exa.zip

# Install bat (better cat) manually
RUN curl -L "https://github.com/sharkdp/bat/releases/download/v0.24.0/bat-v0.24.0-aarch64-unknown-linux-gnu.tar.gz" -o bat.tar.gz \
    && tar -xzf bat.tar.gz --strip-components=1 -C /usr/local/bin/ bat-v0.24.0-aarch64-unknown-linux-gnu/bat \
    && rm bat.tar.gz

# Install ripgrep from GitHub releases (newer version)
RUN curl -L "https://github.com/BurntSushi/ripgrep/releases/download/14.1.0/ripgrep-14.1.0-aarch64-unknown-linux-gnu.tar.gz" -o ripgrep.tar.gz \
    && tar -xzf ripgrep.tar.gz --strip-components=1 -C /usr/local/bin/ ripgrep-14.1.0-aarch64-unknown-linux-gnu/rg \
    && rm ripgrep.tar.gz

# Note: ast-grep skipped for now due to ARM64 binary issues
# Can be added manually later with: cargo install ast-grep

# Install Node.js (LTS)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install additional development tools via npm
RUN npm install -g \
    prettier \
    eslint \
    typescript \
    tsx \
    nodemon \
    http-server \
    json-server \
    tldr

# Install useful Python packages
RUN pip3 install --no-cache-dir \
    httpie \
    ipython \
    black \
    ruff \
    pylint \
    pyyaml \
    requests \
    rich \
    typer

# Create non-root user for development
RUN useradd -m -s /bin/zsh -u 1000 vibeman

# Install oh-my-zsh for vibeman user
USER vibeman
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended

# Install zsh plugins
RUN git clone https://github.com/zsh-users/zsh-autosuggestions ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions \
    && git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting \
    && git clone https://github.com/zsh-users/zsh-completions ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-completions

# Create necessary directories
RUN mkdir -p ~/.claude ~/.config

# Switch back to root for remaining setup
USER root

# Create workspace directories
RUN mkdir -p /workspace /logs /config \
    && chown -R vibeman:vibeman /workspace /logs /config

# Create fd symlink for consistency
RUN ln -s /usr/bin/fdfind /usr/local/bin/fd

# Copy configuration files
COPY --chown=vibeman:vibeman claude-global.md /home/vibeman/.claude/CLAUDE.md
COPY --chown=vibeman:vibeman zshrc /home/vibeman/.zshrc

# Install Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | bash

# Set up Git globals (will be overridden by mounted config)
RUN git config --global init.defaultBranch main \
    && git config --global safe.directory /workspace

# Switch to non-root user
USER vibeman
WORKDIR /workspace

# Environment variables
ENV VIBEMAN_AI_CONTAINER=true \
    TERM=xterm-256color \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    SHELL=/bin/zsh

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD test -d /workspace && test -d /logs || exit 1

# Switch back to root to create entrypoint script
USER root

# Create entrypoint script (using /bin/bash for the script itself is fine)
RUN echo '#!/bin/bash\n\
echo "╔══════════════════════════════════════════════════════════════╗"\n\
echo "║           Vibeman AI Container Started! 🚀                   ║"\n\
echo "╠══════════════════════════════════════════════════════════════╣"\n\
echo "║ To start Claude CLI, run:                                    ║"\n\
echo "║   docker exec -it <container-name> claude --dangerously-skip-permissions ║"\n\
echo "║                                                              ║"\n\
echo "║ Or attach to the container:                                  ║"\n\
echo "║   docker attach <container-name>                             ║"\n\
echo "╚══════════════════════════════════════════════════════════════╝"\n\
echo ""\n\
echo "Container is running. Use docker exec to interact with it."\n\
echo "Logs available in: /logs"\n\
echo "Workspace mounted at: /workspace"\n\
echo ""\n\
# Keep container running indefinitely\n\
while true; do\n\
  sleep 3600\n\
done' > /usr/local/bin/entrypoint.sh && \
    chmod +x /usr/local/bin/entrypoint.sh

# Switch back to vibeman user for runtime
USER vibeman

# Default command - keep container running
CMD ["/usr/local/bin/entrypoint.sh"]