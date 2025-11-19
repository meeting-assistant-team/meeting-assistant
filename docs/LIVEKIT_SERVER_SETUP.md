# LiveKit Server Setup Guide

## Option 1: LiveKit Cloud (Khuyến nghị - FREE tier)

### Ưu điểm:
- ✅ FREE tier: 10,000 phút/tháng (đủ cho dev + demo)
- ✅ Không cần setup server
- ✅ Sẵn sàng ngay lập tức
- ✅ Có Dashboard quản lý
- ✅ Tự động scale
- ✅ Built-in monitoring

### Cách setup:

1. **Đăng ký tài khoản**:
   - Truy cập: https://cloud.livekit.io/
   - Đăng ký free account (có thể dùng Google/GitHub)

2. **Tạo project**:
   - Sau khi login, tạo project mới
   - Đặt tên: `meeting-assistant-dev`

3. **Lấy credentials**:
   - Vào Settings → API Keys
   - Bạn sẽ thấy:
     ```
     LiveKit URL: wss://your-project.livekit.cloud
     API Key: APIxxxxxxxxx
     API Secret: xxxxxxxxxxxxxxxxxxxxxxxx
     ```

4. **Cập nhật .env**:
   ```bash
   # LiveKit Configuration
   LIVEKIT_URL=wss://your-project.livekit.cloud  # Thay bằng URL của bạn
   LIVEKIT_API_KEY=APIxxxxxxxxx                   # Thay bằng API Key của bạn
   LIVEKIT_API_SECRET=xxxxxxxxxxxxxxxxxxxxxxxx    # Thay bằng API Secret của bạn
   LIVEKIT_USE_MOCK=false                         # ⚠️ ĐỔI THÀNH false
   ```

5. **Restart server**:
   ```bash
   make run
   ```

6. **Test với LiveKit Meet**:
   ```bash
   # 1. Gọi API tạo room
   curl -X POST http://localhost:8080/api/v1/rooms \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "title": "Test Meeting",
       "description": "Testing LiveKit Cloud"
     }'
   
   # Response sẽ có:
   {
     "livekit_token": "eyJhbGc...",
     "livekit_url": "wss://your-project.livekit.cloud"
   }
   
   # 2. Truy cập LiveKit Meet với token
   https://meet.livekit.io/custom?liveKitUrl=wss://your-project.livekit.cloud&token=eyJhbGc...
   ```

### Free Tier Limits:
- 10,000 minutes/month (~ 166 giờ)
- Unlimited rooms
- Unlimited participants
- Recording included
- Đủ cho development + demo thesis

---

## Option 2: Self-hosted LiveKit Server (Cho production)

### Ưu điểm:
- ✅ Full control
- ✅ Không giới hạn sử dụng
- ✅ Data ở server riêng
- ❌ Phải setup infrastructure
- ❌ Phải maintain
- ❌ Chi phí server VPS/Cloud

### Requirements:
- VPS/Cloud Server (1 CPU, 2GB RAM tối thiểu)
- Ubuntu 20.04/22.04 hoặc Docker
- Public IP + Domain (optional nhưng khuyến nghị)
- Port 7880 (HTTP), 7881 (HTTPS), 50000-60000/UDP (WebRTC)

### Setup với Docker (Đơn giản nhất):

1. **SSH vào server**:
   ```bash
   ssh user@your-server-ip
   ```

2. **Cài Docker** (nếu chưa có):
   ```bash
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh
   sudo usermod -aG docker $USER
   ```

3. **Tạo file cấu hình**:
   ```bash
   mkdir -p ~/livekit
   cd ~/livekit
   nano livekit.yaml
   ```

   ```yaml
   port: 7880
   bind_addresses:
     - "0.0.0.0"
   
   rtc:
     port_range_start: 50000
     port_range_end: 60000
     use_external_ip: true
   
   keys:
     APIxxxxxxxxxxx: your-secret-key-here  # Thay bằng key tự generate
   
   logging:
     level: info
   ```

4. **Generate API Key/Secret**:
   ```bash
   # Cài livekit-cli
   curl -sSL https://get.livekit.io/cli | bash
   
   # Generate keys
   livekit-cli create-token --api-key APIxxxxxxxxxxx --api-secret your-secret-key-here
   ```

5. **Chạy LiveKit Server**:
   ```bash
   docker run -d \
     --name livekit \
     --restart unless-stopped \
     -p 7880:7880 \
     -p 7881:7881 \
     -p 50000-60000:50000-60000/udp \
     -v $(pwd)/livekit.yaml:/livekit.yaml \
     livekit/livekit-server \
     --config /livekit.yaml
   ```

6. **Kiểm tra server đang chạy**:
   ```bash
   docker logs livekit
   # Phải thấy: "starting LiveKit server"
   
   curl http://your-server-ip:7880
   # Phải response: LiveKit Server
   ```

7. **Cập nhật .env** (backend của bạn):
   ```bash
   LIVEKIT_URL=ws://your-server-ip:7880  # hoặc wss:// nếu có SSL
   LIVEKIT_API_KEY=APIxxxxxxxxxxx
   LIVEKIT_API_SECRET=your-secret-key-here
   LIVEKIT_USE_MOCK=false
   ```

### Setup SSL với Nginx (Production):

```bash
# Cài Nginx + Certbot
sudo apt install nginx certbot python3-certbot-nginx

# Tạo Nginx config
sudo nano /etc/nginx/sites-available/livekit

# Config:
server {
    server_name livekit.yourdomain.com;
    
    location / {
        proxy_pass http://localhost:7880;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}

# Enable site
sudo ln -s /etc/nginx/sites-available/livekit /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Get SSL certificate
sudo certbot --nginx -d livekit.yourdomain.com

# Sau đó URL sẽ là:
# LIVEKIT_URL=wss://livekit.yourdomain.com
```

---

## Option 3: Local LiveKit Server (Development only)

### Cho testing trên máy local:

1. **Download LiveKit Server**:
   ```bash
   # macOS
   brew install livekit
   
   # Hoặc download binary
   # https://github.com/livekit/livekit/releases
   ```

2. **Tạo config**:
   ```bash
   mkdir -p ~/livekit-local
   cd ~/livekit-local
   
   cat > livekit.yaml <<EOF
   port: 7880
   rtc:
     port_range_start: 50000
     port_range_end: 50100
   keys:
     devkey: secret
   EOF
   ```

3. **Chạy server**:
   ```bash
   livekit-server --config livekit.yaml
   ```

4. **Cập nhật .env**:
   ```bash
   LIVEKIT_URL=ws://localhost:7880
   LIVEKIT_API_KEY=devkey
   LIVEKIT_API_SECRET=secret
   LIVEKIT_USE_MOCK=false
   ```

⚠️ **Chỉ dùng cho dev trên máy local, không public được!**

---

## So sánh các options:

| Feature | LiveKit Cloud | Self-hosted | Local |
|---------|--------------|-------------|-------|
| Setup Time | 5 phút | 1-2 giờ | 10 phút |
| Chi phí | Free (10K phút) | $5-20/tháng VPS | Free |
| Scalability | Auto | Manual | Không |
| SSL/Security | Built-in | Tự setup | Không |
| Production | ✅ | ✅ | ❌ |
| Development | ✅ | ✅ | ✅ |
| Public Access | ✅ | ✅ | ❌ |
| Khuyến nghị | **Dùng cho thesis** | Production sau này | Test local only |

---

## Khuyến nghị cho bạn:

### Phase 1: Development & Thesis Demo (HIỆN TẠI)
👉 **Dùng LiveKit Cloud (Free)**
- Setup trong 5 phút
- Free 10,000 phút/tháng (đủ cho demo thesis)
- Không cần setup server
- Có thể test real meeting ngay

### Phase 2: Production (Sau khi bảo vệ)
👉 **Self-hosted trên VPS**
- Full control
- Không giới hạn
- Customize được

---

## Test sau khi setup LiveKit Server:

```bash
# 1. Kiểm tra backend connect được LiveKit
make run
# Phải thấy log: "✅ LiveKit connected successfully"

# 2. Tạo room qua API
curl -X POST http://localhost:8080/api/v1/rooms \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "Real Meeting Test",
    "description": "Testing with real LiveKit"
  }'

# 3. Copy token từ response, mở LiveKit Meet:
https://meet.livekit.io/custom?liveKitUrl=wss://your-livekit.cloud&token=YOUR_TOKEN

# 4. Mở thêm tab/máy khác với cùng room token
# → Bạn sẽ thấy 2 người trong meeting!
```

---

## Next Steps:

1. ✅ Chọn option (khuyến nghị: LiveKit Cloud cho thesis)
2. ✅ Setup theo hướng dẫn trên
3. ✅ Cập nhật .env với credentials thật
4. ✅ Set `LIVEKIT_USE_MOCK=false`
5. ✅ Restart backend
6. ✅ Test tạo room và join meeting thật

Bạn muốn tôi hướng dẫn chi tiết setup LiveKit Cloud không? (Nhanh nhất, 5 phút là xong!)
