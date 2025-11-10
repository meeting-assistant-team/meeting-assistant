# 🚀 Deployment Guide - GitHub Actions SSH Deploy with Docker

## 📋 Quy trình Deploy (Step-by-step Checklist)

### **PHASE 1: Chuẩn bị Server** ✅/❌

- [ ] **1.1. Thuê/Chuẩn bị VPS**
  - Ubuntu 20.04+ hoặc Debian 11+
  - Tối thiểu: 2GB RAM, 2 CPU cores, 20GB storage
  - Public IP address

- [ ] **1.2. Cài đặt Docker trên server**
  ```bash
  # SSH vào server
  ssh username@your-server-ip
  
  # Update system
  sudo apt update && sudo apt upgrade -y
  
  # Install Docker
  curl -fsSL https://get.docker.com -o get-docker.sh
  sudo sh get-docker.sh
  
  # Add user to docker group
  sudo usermod -aG docker $USER
  
  # Install Docker Compose
  sudo apt install docker-compose-plugin -y
  
  # Verify installation
  docker --version
  docker compose version
  ```

- [ ] **1.3. Setup Firewall**
  ```bash
  # Allow SSH, HTTP, HTTPS
  sudo ufw allow 22/tcp
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  sudo ufw enable
  
  # Check status
  sudo ufw status
  ```

- [ ] **1.4. Tạo SSH Key cho GitHub Actions**
  ```bash
  # Trên server, tạo SSH key
  ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions
  
  # Add public key vào authorized_keys
  cat ~/.ssh/github_actions.pub >> ~/.ssh/authorized_keys
  
  # Copy private key (sẽ dùng cho GitHub Secrets)
  cat ~/.ssh/github_actions
  # ⚠️ COPY TOÀN BỘ OUTPUT (bao gồm cả -----BEGIN/END-----)
  ```

---

### **PHASE 2: Chuẩn bị Code & Files** ✅/❌

- [ ] **2.1. Tạo `.gitignore`**
  - File đã được tạo tự động
  - Đảm bảo `.env`, `tmp/`, `bin/` không được commit

- [ ] **2.2. Tạo `Dockerfile` cho production**
  - File cần được tạo: `Dockerfile` (hoặc `Dockerfile.prod`)

- [ ] **2.3. Tạo `docker-compose.prod.yml`**
  - File cần được tạo cho production environment

- [ ] **2.4. Tạo `.dockerignore`**
  - File cần được tạo để giảm image size

---

### **PHASE 3: Setup GitHub Repository** ✅/❌

- [ ] **3.1. Push code lên GitHub**
  ```bash
  # Tại thư mục project
  git init
  git add .
  git commit -m "Initial commit"
  git branch -M main
  git remote add origin https://github.com/johnquangdev/speakup.git
  git push -u origin main
  ```

- [ ] **3.2. Thêm GitHub Secrets**
  - Vào repository → Settings → Secrets and variables → Actions → New repository secret
  
  **Required Secrets:**
  
  | Secret Name | Giá trị | Ví dụ |
  |-------------|---------|-------|
  | `SSH_HOST` | IP address server | `123.45.67.89` |
  | `SSH_USERNAME` | Username SSH | `ubuntu` hoặc `root` |
  | `SSH_PRIVATE_KEY` | Private key từ bước 1.4 | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
  | `DB_PASSWORD` | PostgreSQL password | `your-strong-password-123` |
  | `REDIS_PASSWORD` | Redis password (optional) | `redis-password-123` |
  | `JWT_SECRET` | JWT secret key | `super-secret-jwt-key-change-this` |
  | `GOOGLE_CLIENT_ID` | Google OAuth Client ID | `123456789-abc.apps.googleusercontent.com` |
  | `GOOGLE_CLIENT_SECRET` | Google OAuth Secret | `GOCSPX-...` |
  | `GOOGLE_REDIRECT_URL` | OAuth Redirect URL | `https://yourdomain.com/v1/auth/google/callback` |
  | `CORS_ALLOWED_ORIGINS` | Allowed origins | `https://yourdomain.com,https://www.yourdomain.com` |

  **Cách thêm secrets:**
  1. Copy giá trị secret
  2. Vào GitHub repo → Settings → Secrets and variables → Actions
  3. Click "New repository secret"
  4. Paste tên và giá trị
  5. Click "Add secret"

---

### **PHASE 4: Tạo Production Files** ✅/❌

- [ ] **4.1. Tạo `Dockerfile`**
  ```dockerfile
  # Multi-stage build để giảm image size
  FROM golang:1.21-alpine AS builder
  
  WORKDIR /app
  
  # Copy go mod files
  COPY go.mod go.sum ./
  RUN go mod download
  
  # Copy source code
  COPY . .
  
  # Build binary
  RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api
  
  # Final stage
  FROM alpine:latest
  
  RUN apk --no-cache add ca-certificates
  
  WORKDIR /root/
  
  # Copy binary from builder
  COPY --from=builder /app/main .
  
  EXPOSE 8080
  
  CMD ["./main"]
  ```

- [ ] **4.2. Tạo `docker-compose.prod.yml`**
  ```yaml
  version: '3.8'
  
  services:
    postgres:
      image: postgres:16-alpine
      container_name: meeting-assistant-postgres-prod
      restart: always
      environment:
        POSTGRES_USER: ${DB_USER}
        POSTGRES_PASSWORD: ${DB_PASSWORD}
        POSTGRES_DB: ${DB_NAME}
      volumes:
        - postgres_data:/var/lib/postgresql/data
      networks:
        - app-network
      healthcheck:
        test: ["CMD-SHELL", "pg_isready -U postgres"]
        interval: 10s
        timeout: 5s
        retries: 5
  
    redis:
      image: redis:7-alpine
      container_name: meeting-assistant-redis-prod
      restart: always
      command: redis-server --requirepass ${REDIS_PASSWORD}
      volumes:
        - redis_data:/data
      networks:
        - app-network
      healthcheck:
        test: ["CMD", "redis-cli", "ping"]
        interval: 10s
        timeout: 5s
        retries: 5
  
    app:
      build:
        context: .
        dockerfile: Dockerfile
      container_name: meeting-assistant-app-prod
      restart: always
      ports:
        - "8080:8080"
      env_file:
        - .env
      depends_on:
        postgres:
          condition: service_healthy
        redis:
          condition: service_healthy
      networks:
        - app-network
      healthcheck:
        test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
        interval: 30s
        timeout: 10s
        retries: 3
  
  networks:
    app-network:
      driver: bridge
  
  volumes:
    postgres_data:
    redis_data:
  ```

- [ ] **4.3. Tạo `.dockerignore`**
  ```
  .git
  .github
  .env
  .env.example
  .air.toml
  tmp/
  bin/
  *.md
  docs/
  .gitignore
  Makefile
  .vscode
  .idea
  ```

---

### **PHASE 5: Setup Domain & SSL (Optional nhưng khuyến nghị)** ✅/❌

- [ ] **5.1. Point domain về server**
  - Cấu hình DNS A record trỏ domain về IP server

- [ ] **5.2. Cài đặt Nginx reverse proxy**
  ```bash
  # Trên server
  sudo apt install nginx -y
  
  # Tạo config cho domain
  sudo nano /etc/nginx/sites-available/meeting-assistant
  ```
  
  ```nginx
  server {
      listen 80;
      server_name yourdomain.com www.yourdomain.com;
      
      location / {
          proxy_pass http://localhost:8080;
          proxy_http_version 1.1;
          proxy_set_header Upgrade $http_upgrade;
          proxy_set_header Connection 'upgrade';
          proxy_set_header Host $host;
          proxy_cache_bypass $http_upgrade;
          proxy_set_header X-Real-IP $remote_addr;
          proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto $scheme;
      }
  }
  ```
  
  ```bash
  # Enable site
  sudo ln -s /etc/nginx/sites-available/meeting-assistant /etc/nginx/sites-enabled/
  sudo nginx -t
  sudo systemctl restart nginx
  ```

- [ ] **5.3. Cài đặt SSL với Certbot**
  ```bash
  sudo apt install certbot python3-certbot-nginx -y
  sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com
  ```

---

### **PHASE 6: Deploy & Verify** ✅/❌

- [ ] **6.1. Trigger deployment**
  ```bash
  # Push code để trigger GitHub Actions
  git add .
  git commit -m "Setup deployment"
  git push origin main
  ```

- [ ] **6.2. Monitor deployment**
  - Vào GitHub repo → Actions tab
  - Xem workflow "Deploy to Production Server" đang chạy
  - Check logs từng step

- [ ] **6.3. Verify trên server**
  ```bash
  # SSH vào server
  ssh username@your-server-ip
  
  # Check containers đang chạy
  cd ~/meeting-assistant
  docker compose -f docker-compose.prod.yml ps
  
  # Check logs
  docker compose -f docker-compose.prod.yml logs -f app
  
  # Test health endpoint
  curl http://localhost:8080/health
  ```

- [ ] **6.4. Test OAuth flow**
  - Truy cập: `https://yourdomain.com/v1/auth/google/login`
  - Đăng nhập Google
  - Verify callback hoạt động

---

## 🔄 Deployment Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    DEVELOPER                                │
│  git push origin main                                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                 GITHUB ACTIONS                              │
│  1. Checkout code                                           │
│  2. Setup Docker Buildx                                     │
│  3. Create .env on server (via SSH)                         │
│  4. Deploy with Docker Compose (via SSH)                    │
│  5. Health Check                                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                   SERVER (VPS)                              │
│  1. Pull latest code from GitHub                           │
│  2. Stop old containers                                     │
│  3. Build new Docker images                                 │
│  4. Start new containers (app, postgres, redis)             │
│  5. Run migrations                                          │
│  6. Serve traffic on port 8080                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 🐛 Troubleshooting

### Lỗi thường gặp:

**1. SSH connection refused**
```bash
# Kiểm tra SSH service trên server
sudo systemctl status ssh

# Restart SSH service
sudo systemctl restart ssh
```

**2. Docker permission denied**
```bash
# Add user vào docker group
sudo usermod -aG docker $USER
newgrp docker
```

**3. Port 8080 already in use**
```bash
# Kill process đang dùng port 8080
sudo lsof -ti:8080 | xargs kill -9
```

**4. Database connection failed**
```bash
# Check PostgreSQL container logs
docker logs meeting-assistant-postgres-prod

# Check network
docker network inspect meeting-assistant_app-network
```

---

## 📝 Post-Deployment Checklist

- [ ] Monitoring setup (Prometheus/Grafana)
- [ ] Backup strategy cho PostgreSQL
- [ ] Log aggregation (ELK/Loki)
- [ ] Auto-scaling setup (nếu cần)
- [ ] Update Google OAuth redirect URL về production domain
- [ ] Setup alerts cho downtime

---

## 📞 Support

Nếu gặp vấn đề, check:
1. GitHub Actions logs
2. Server logs: `docker compose -f docker-compose.prod.yml logs`
3. Application logs trong container

**Current Progress:** ☐☐☐☐☐☐ 0/6 Phases Complete
