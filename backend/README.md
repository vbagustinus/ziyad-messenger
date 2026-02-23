# Ziyad Messenger Backend API Documentation

Daftar API yang tersedia di backend untuk dikonsumsi oleh client.

---

## 🚀 Alur Komunikasi (Slack-Style)

### 1. Obrolan Grup (Group Chat)

1. **Daftarkan Channel**: Gunakan Admin Dashboard atau Admin API (`POST /admin/channels`).
2. **Kelola Anggota**: Tambahkan user ke channel (`POST /admin/channels/{id}/members`).
3. **Kirim/Terima**: Client konek ke WebSocket Messaging Service (`/ws?user_id=...`). Server hanya akan mengirim pesan grup ke user yang terdaftar sebagai anggota.

### 2. Pesan Langsung (Direct Message)

1. **Langsung Kirim**: Client mengirim pesan via WebSocket dengan `channel_id` berisi **UserID** tujuan.
2. **Otomatisasi**: Backend akan otomatis membuat channel privat `dm:userA:userB` dan mendaftarkan kedua user tersebut.
3. **Privasi**: Pesan DM hanya bisa dilihat dan diterima oleh kedua user tersebut.

### 3. Pengiriman File (File Transfer)

1. **Upload**: Client POST file ke `/upload` (Port 8082). Mendapatkan `file_id` dan `key`.
2. **Notifikasi**: Client mengirim pesan chat biasa dengan `type: 3` (File) dan isi pesan berupa JSON metadata file tersebut.
3. **Download**: Penerima mengambil file via `/download?id={file_id}`.

---

## Detail Service & Port... (lihat di bawah)

## 1. Admin API (Port 8090)

Digunakan khusus untuk Admin Dashboard.

### Authentication & Admin

- `POST /admin/login`: Login admin
- `GET /admin/me`: Ambil profile admin yang login
- `POST /admin/admins`: Buat admin baru

### User Management

- `GET /admin/users`: List semua user
- `POST /admin/users`: Buat user baru
- `PUT /admin/users/{id}`: Edit user
- `DELETE /admin/users/{id}`: Hapus user
- `POST /admin/users/{id}/reset-password`: Reset password user

### Monitoring & System

- `GET /admin/monitoring/network`: Status jaringan
- `GET /admin/monitoring/users`: Statistik user online
- `GET /admin/monitoring/messages`: Statistik pesan
- `GET /admin/monitoring/files`: Statistik transfer file
- `GET /admin/monitoring/system`: Info sistem server
- `GET /admin/system/health`: Cek kesehatan sistem
- `GET /admin/cluster/status`: Status cluster node

---

## 2. Auth Service (Port 8086)

Digunakan untuk registrasi dan login user umum.

- `POST /login`: Login user (Payload: `username`, `password`)
- `POST /register`: Registrasi user baru (Payload: `username`, `full_name`, `password`, `role`)
- `GET /health`: Cek status service

---

## 3. Messaging Service (Port 8081)

Layanan utama untuk pengiriman pesan.

- `GET /ws?user_id={id}`: Koneksi WebSocket untuk real-time chat
- `GET /history?channel_id={id}`: Ambil riwayat pesan dalam channel
- `POST /send`: Kirim pesan via HTTP (Alternatif WebSocket)
- `GET /health`: Cek status service

---

## 4. File Transfer Service (Port 8082)

Layanan untuk upload dan download file.

- `POST /upload`: Upload file (Multipart form)
- `GET /download?id={file_id}`: Download file berdasarkan ID
- `GET /health`: Cek status service

---

## 5. Presence Service (Port 8083)

Layanan untuk status online/offline user.

- `POST /heartbeat`: Update status online user (Payload: `user_id`, `status`)
- `GET /status?user_id={id}`: Cek status terkini user
- `GET /health`: Cek status service

---

## 6. Audit Service (Port 8084)

Layanan pencatatan aktivitas sistem.

- `POST /log`: Mencatat event audit baru
- `GET /log`: (Internal/Admin) Mengambil log audit

---

## 7. Cluster Service (Port 8085)

Layanan koordinasi antar node backend.

- `GET /status`: Status kesehatan cluster
