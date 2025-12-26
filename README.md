# Simple TinyURL Service

Service API URL shortener sederhana yang dibuat menggunakan Go dan Redis.

## Tech Stack

- **Go** (Golang)
- **Redis** (untuk penyimpanan data sementara 24 jam)
- **Docker** & **Docker Compose**

## Cara Menjalankan

### Menggunakan Docker (Recomended)

Jalankan perintah berikut untuk menjalankan aplikasi dan Redis sekaligus:

```bash
docker-compose up --build
```

Aplikasi akan berjalan di `http://localhost:8080`.

### Menjalankan Secara Manual (Local)

Pastikan Anda memiliki instance Redis yang berjalan, kemudian:

1.  Set environment variable (opsional jika default):
    ```bash
    export REDIS_ADDR=localhost:6379
    export SERVER_URL=http://localhost:8080
    ```
2.  Jalankan aplikasi:
    ```bash
    go run .
    ```

## API Endpoints

### 1. Membuat Short URL

Membuat link pendek baru dari URL panjang. Link akan kadaluarsa dalam 24 jam.

- **URL**: `/tinyurl`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Body**:

```json
{
  "long_url": "https://www.google.com/very/long/url/path"
}
```

- **Response**:

```json
{
  "short_url": "http://localhost:8080/aBcD123456",
  "long_url": "https://www.google.com/very/long/url/path",
  "message": "Short URL will be expired in 24 hours"
}
```

### 2. Redirect URL

Mengakses short URL akan me-redirect pengguna ke URL asli.

- **URL**: `/{kode_unik}` (contoh: `/aBcD123456`)
- **Method**: `GET`
- **Response**: 303 See Other (Redirect ke URL asli)

## Konfigurasi

| Variable | Deskripsi | Default |
|----------|-----------|---------|
| `REDIS_ADDR` | Alamat koneksi ke Redis | `localhost:6379` |
| `SERVER_URL` | Base URL server untuk prefix short URL | `http://localhost:8080` |
