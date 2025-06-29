# Self-hosted GitHub Actions Runner for TurboScript
FROM ubuntu:22.04

# Avoid tzdata interactive prompt
ENV DEBIAN_FRONTEND=noninteractive

# Install required packages
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    jq \
    tar \
    unzip \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release \
    sudo \
    build-essential \
    libssl-dev \
    libffi-dev \
    python3 \
    python3-pip \
    bc \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Install Docker CLI (for Docker-in-Docker operations)
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
RUN echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt-get update && apt-get install -y docker-ce-cli && rm -rf /var/lib/apt/lists/*

# Install Docker Compose
RUN curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
RUN chmod +x /usr/local/bin/docker-compose

# Install Go 1.23.0
RUN wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
RUN rm go1.23.0.linux-amd64.tar.gz

# Install Node.js 20 (LTS)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
RUN apt-get install -y nodejs

# Install Newman for Postman testing
RUN npm install -g newman newman-reporter-html

# Install hey for load testing (with CGO disabled to avoid gcc issues)
RUN CGO_ENABLED=0 /usr/local/go/bin/go install github.com/rakyll/hey@latest

# Set environment variables for root user initially
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"
ENV GOPATH="/root/go"
ENV GOOS="linux"
ENV GOARCH="amd64"

# Create runner user
RUN useradd -m -s /bin/bash runner
RUN usermod -aG sudo runner
RUN echo "runner ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Create docker group and add runner to it for Docker socket access
RUN groupadd -f docker || true
RUN usermod -aG docker runner

# Set up Go environment for runner user
RUN mkdir -p /home/runner/go/bin /home/runner/go/pkg /home/runner/go/src
RUN chown -R runner:runner /home/runner/go

# Switch to runner user
USER runner
WORKDIR /home/runner

# Set Go environment variables for runner user
ENV PATH="/usr/local/go/bin:/home/runner/go/bin:${PATH}"
ENV GOPATH="/home/runner/go"
ENV GOOS="linux"
ENV GOARCH="amd64"

# Download and install GitHub Actions runner
ARG RUNNER_VERSION="2.319.1"
RUN curl -o actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz -L https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz
RUN tar xzf actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz
RUN rm actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz

# Install runner dependencies
RUN sudo ./bin/installdependencies.sh

# Install additional Go tools needed for CI (with CGO disabled to avoid gcc issues)
RUN CGO_ENABLED=0 /usr/local/go/bin/go install github.com/securego/gosec/v2/cmd/gosec@latest
RUN CGO_ENABLED=0 /usr/local/go/bin/go install github.com/sonatype-nexus-community/nancy@latest
RUN CGO_ENABLED=0 /usr/local/go/bin/go install golang.org/x/vuln/cmd/govulncheck@latest

# Create scripts directory
RUN mkdir -p /home/runner/scripts

# Copy startup script
COPY --chown=runner:runner start-runner.sh /home/runner/scripts/
RUN chmod +x /home/runner/scripts/start-runner.sh

# Set working directory for runner
WORKDIR /home/runner

# Expose any ports if needed (optional)
EXPOSE 80 443

# Default command
CMD ["/home/runner/scripts/start-runner.sh"]
