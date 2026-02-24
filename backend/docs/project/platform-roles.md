# Project Role Charter (Untuk Transisi AI)

## Inisiatif Utama

**Enterprise-grade LAN-first communication platform**

Target produk: platform komunikasi internal enterprise yang aman, real-time, self-hosted, dan tetap operasional di jaringan lokal/offline.

## 1) Backend Platform

- **Peran utama**
  - Pemilik domain core system dan source of truth data.
- **Tanggung jawab**
  - Auth, messaging, presence, file transfer, audit, discovery, dan cluster coordination.
  - Kontrak API/protocol (HTTP, WebSocket, UDP discovery) dan kompatibilitas antarlayanan.
  - Security backend: JWT, password hashing, akses berbasis role, audit trail.
  - Keandalan service: health check, isolasi failure, persistence data.
- **Output yang harus dijaga**
  - API stabil, schema konsisten, dan behavior layanan tidak regress untuk client/admin.

## 2) Frontend Admin Platform

- **Peran utama**
  - Pemilik control plane untuk administrator sistem.
- **Tanggung jawab**
  - Login admin, manajemen user/role/department/channel/device.
  - Monitoring operasional, sistem status, dan visibilitas audit.
  - Konsumsi API admin secara aman dan konsisten dengan kontrak backend.
- **Output yang harus dijaga**
  - Dashboard operasional yang jelas, cepat, dan aman untuk workflow admin harian.

## 3) Client Platform (Desktop/Mobile)

- **Peran utama**
  - Pemilik pengalaman end-user untuk komunikasi harian.
- **Tanggung jawab**
  - Login user, kirim/terima pesan realtime, riwayat chat, presence, alur file.
  - Transport realtime (WebSocket) dengan fallback (HTTP) bila perlu.
  - Manajemen session/token lokal dan konfigurasi endpoint lintas environment LAN.
- **Output yang harus dijaga**
  - UX chat stabil, latency rendah, dan perilaku konsisten saat jaringan tidak ideal.

## 4) Operations Ownership (Per Platform)

- **Backend**
  - Menjaga `backend/deploy/*` (compose, Dockerfile, helm, runbook operasi backend).
- **Frontend Admin**
  - Menjaga `frontend/admin-dashboard/deploy/*` (build/run dashboard).
- **Client**
  - Menjaga `clients/desktop-app/deploy/*` (packaging/distribution notes).
- **Output yang harus dijaga**
  - Tiap platform punya jalur deploy sendiri, saling terhubung via API, dan tidak saling menumpuk ownership tooling.

## Aturan Sinkronisasi Antar Platform

- Backend mengubah kontrak API/protocol hanya dengan dampak analysis ke Frontend/Client.
- Frontend dan Client tidak membuat asumsi endpoint di luar kontrak resmi backend.
- Tiap platform wajib sinkron kebutuhan runtime sebelum rilis lintas platform.
- Semua AI pengganti harus membaca dokumen ini dulu sebelum melakukan perubahan lintas platform.
