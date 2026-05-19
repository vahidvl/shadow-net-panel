<h1 align="center">
  <img src="https://raw.githubusercontent.com/MahMotion/3x-ui-shadownet/main/frontend/src/assets/logo.png" alt="3x-ui Shadow-Net" width="120" />
  <br>
  3x-ui: Shadow-Net Edition
</h1>

<p align="center">
  <b>A stealth-focused, commercial-grade fork of the official 3x-ui panel, engineered for maximum resilience and multi-bot automation.</b>
</p>

<p align="center">
  <a href="README-fa.md">🇮🇷 راهنمای فارسی</a> | <a href="README.md">🇺🇸 English Guide</a>
</p>

---

## 🌟 Overview

**Shadow-Net Edition** is a heavily customized, production-ready fork of the widely popular [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui) v3.0.2. It introduces advanced stealth routing, UI/UX enhancements, and deep integration hooks for Telegram automation bots, making it the perfect backend for VPN sellers and commercial operators.

Unlike typical modifications, Shadow-Net integrates natively into the Go backend and Vue frontend, ensuring zero performance penalty while adding critical business features.

## 🚀 Exclusive Features

1. **Smart Outbound Proxy Bridge**
   - Renders local API/Telegram traffic out through a dynamically configured upstream proxy (SOCKS5/HTTP/VLESS).
   - Bypasses local datacenter censorship (e.g., ArvanCloud/Iran restrictions) safely without breaking Xray's global routing.

2. **Pre-Flight Handshake Validation**
   - Automatically tests proxy configurations against Cloudflare DoH (`1.1.1.1`) on an isolated temporary socket before activation.
   - Eliminates "false positive" running states in the UI.

3. **Triple-Bot Infrastructure Hooks**
   - Native database integration and settings tabs for three independent Telegram bots: **Sales Bot**, **Sentinel Bot** (Monitoring), and **Admin Bot**.
   - Tokens are securely redacted from frontend public API views.

4. **Three-Strikes Watchdog (Penalty System)**
   - Client-level IP monitoring and a `penalty` tracking system to enforce account usage limits and automatically disable abusers.

5. **Premium UI/UX Enhancements**
   - **Proxy Control Panel:** An elegant, theme-aware control pane located above the Logout button for instant proxy toggling.
   - **Pulsing Header Badge:** A minimal, animated status indicator globally visible in the header when the proxy bridge is active.

---

## 💻 Installation & Upgrade

We provide a seamless, automated installation script that upgrades any existing 3x-ui panel to the Shadow-Net Edition without data loss.

```bash
bash <(curl -Ls https://raw.githubusercontent.com/YOUR_GITHUB_USERNAME/shadow-net-panel/main/patch_v3.0.2_shadow_net.sh)
```
> **Note:** Replace `YOUR_GITHUB_USERNAME` with your actual GitHub username once pushed.

The installer supports two modes:
- **Mode 1 (Fast):** Downloads a precompiled `x-ui` binary and replaces your current panel instantly.
- **Mode 2 (Source):** Pulls the source code, compiles the Vue frontend and Go backend locally on your server, and installs it.

---

## 🛠️ Building from Source

If you prefer to compile the panel yourself for security auditing or custom architecture (e.g., ARM64):

### Prerequisites
- [Go](https://go.dev/doc/install) >= 1.22
- [Node.js](https://nodejs.org/en/download/) >= 22 (and `npm` >= 10)

### 1. Build the Frontend (Vite/Vue)
```bash
cd frontend
npm install
npm run build
cd ..
```

### 2. Compile the Go Backend
```bash
go build -tags=jsoniter -ldflags="-s -w" -o bin/x-ui main.go
```

The resulting `x-ui` binary inside the `bin/` directory contains the full application (with the frontend embedded).

---

## 🤝 Upstream Acknowledgment

This project is built upon the phenomenal work of [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui). All core Xray routing and multi-node functionality belongs to the original authors. Shadow-Net focuses purely on commercial and stealth extensions.

## 📝 License
This project is licensed under the GPL-3.0 License - see the LICENSE file for details.
