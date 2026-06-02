# SelfVOD — Self-Hosted Video on Demand Service

A lightweight, self-hosted Video on Demand microservice built with **Go** and **Garage S3**. Designed as a drop-in replacement for managed video services like Bunny Stream, giving you full control over your media storage and delivery.

## Features

- **Folder Management** — Organize videos into folders for clean categorization
- **Multipart Video Upload** — Upload large video files (up to 500 MB) directly to S3-compatible storage
- **Presigned Streaming URLs** — Time-limited, cryptographically signed URLs for secure video playback
- **Auto-Refreshing Token** — Client player silently refreshes the stream token before expiry, zero interruption
- **SQLite Metadata** — Zero-config embedded database for video and folder metadata
- **Auth-Free by Design** — No built-in auth; designed for integration via Hasura Actions or reverse proxy auth
- **Docker-Ready** — Single `docker compose up` to run the entire stack

## Architecture

```
┌─────────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Nuxt 3 Frontend   │────▶│   Go API (:8080)  │────▶│ Garage S3 (:3900)│
│  Admin + Client UI  │     │  chi/v5 router    │     │  Private Bucket  │
│  (:3000 dev)        │     │  SQLite metadata  │     │  vod-private     │
└─────────────────────┘     └──────────────────┘     └──────────────────┘
         │                                                    │
         │              Presigned URL (4h TTL)                │
         └────────────────────────────────────────────────────┘
                    Browser streams directly from S3
```

## Project Structure

```
test_vod/
├── cmd/vod/main.go          # Entry point, routing, server setup
├── config/config.go         # Environment-based configuration
├── handler/
│   ├── video.go             # Upload, list, delete, stream handlers
│   └── folder.go            # Folder CRUD handlers
├── s3/client.go             # S3 client wrapper (upload, presign, delete)
├── store/
│   ├── interface.go         # Data models & Store interface
│   └── sqlite.go            # SQLite implementation with migrations
├── frontend/                # Nuxt 3 test frontend
│   ├── pages/
│   │   ├── index.vue        # Landing page
│   │   ├── admin/index.vue  # Admin dashboard (folder/video management)
│   │   └── client/player.vue# Secure video player with token refresh
│   └── nuxt.config.ts       # Nuxt config with API proxy
├── docker-compose.yml       # Full stack: Go API + Garage S3
├── Dockerfile               # Multi-stage Go build
└── README.md
```

## Quick Start

### Prerequisites

- Docker & Docker Compose

### Run

```bash
docker compose up -d
```

This starts:
| Service       | Port  | Description                    |
|---------------|-------|--------------------------------|
| Go API        | 8080  | REST API for video management  |
| Garage S3     | 3901  | S3-compatible object storage   |
| Garage Web    | 3902  | Garage admin web UI            |

### Frontend (Development)

```bash
cd frontend
npm install
npm run dev
```

Then open:
- **Admin Dashboard:** http://localhost:3000/admin
- **Video Player:** http://localhost:3000/client/player?id=VIDEO_ID

## API Reference

All endpoints are under `/api/v1/`.

### Folders

| Method   | Endpoint             | Description          |
|----------|----------------------|----------------------|
| `GET`    | `/api/v1/folders`    | List all folders     |
| `POST`   | `/api/v1/folders`    | Create a folder      |
| `DELETE` | `/api/v1/folders/:id`| Delete a folder      |

**Create Folder:**
```bash
curl -X POST http://localhost:8080/api/v1/folders \
  -H "Content-Type: application/json" \
  -d '{"name": "Course Videos"}'
```

### Videos

| Method   | Endpoint                      | Description                          |
|----------|-------------------------------|--------------------------------------|
| `GET`    | `/api/v1/videos`              | List videos (optional `?folder_id=`) |
| `POST`   | `/api/v1/videos/upload`       | Upload a video (multipart form)      |
| `GET`    | `/api/v1/videos/:id/stream`   | Get presigned stream URL             |
| `DELETE` | `/api/v1/videos/:id`          | Delete video (S3 + metadata)         |

**Upload Video:**
```bash
curl -X POST http://localhost:8080/api/v1/videos/upload \
  -F "file=@video.mp4" \
  -F "title=My Video" \
  -F "folder_id=FOLDER_UUID"
```

**Get Stream URL:**
```bash
curl http://localhost:8080/api/v1/videos/VIDEO_UUID/stream
```

Response:
```json
{
  "video_id": "uuid",
  "title": "My Video",
  "url": "https://s3.example.com/vod-private/videos/...?X-Amz-Signature=...",
  "expires_in": 14400
}
```

### Health

| Method | Endpoint  | Description   |
|--------|-----------|---------------|
| `GET`  | `/health` | Health check  |

## Configuration

All configuration is via environment variables:

| Variable                 | Default                    | Description                          |
|--------------------------|----------------------------|--------------------------------------|
| `VOD_LISTEN_ADDR`        | `:8080`                    | Server listen address                |
| `VOD_DATA_DIR`           | `./data`                   | SQLite database directory            |
| `VOD_S3_ENDPOINT`        | `http://localhost:3901`    | S3 endpoint (internal/Docker)        |
| `VOD_S3_PUBLIC_ENDPOINT` | `http://localhost:3901`    | S3 endpoint (public, for presigning) |
| `VOD_S3_ACCESS_KEY`      | —                          | S3 access key                        |
| `VOD_S3_SECRET_KEY`      | —                          | S3 secret key                        |
| `VOD_S3_REGION`          | `garage`                   | S3 region                            |
| `VOD_S3_BUCKET`          | `vod-private`              | S3 bucket name                       |
| `VOD_S3_USE_PATH_STYLE`  | `true`                     | Use path-style S3 URLs (for Garage)  |
| `VOD_MAX_UPLOAD_SIZE`    | `524288000` (500 MB)       | Maximum upload size in bytes         |
| `VOD_ALLOWED_TYPES`      | `mp4,mov,mkv,webm`        | Comma-separated allowed extensions   |
| `VOD_PRESIGN_EXPIRY`     | `14400` (4 hours)          | Presigned URL lifetime in seconds    |

## How Secure Streaming Works

1. Admin uploads a video → stored in **private** S3 bucket (no public access)
2. Client requests `/api/v1/videos/:id/stream`
3. Backend generates a **presigned URL** signed with S3 credentials, valid for 4 hours
4. Browser loads the video directly from S3 using the signed URL
5. The Vue player auto-refreshes the token at 75% of its lifetime silently
6. After expiry, the URL becomes completely invalid — no access without a fresh token

## Production Notes

- **Nginx/Reverse Proxy:** Use Nginx to map your domain to the Go API (`:8080`) and Garage S3 (`:3900`) ports, and to terminate SSL
- **Hasura Integration:** Register the VOD endpoints as Hasura Actions for authenticated access from your main application
- **Private Buckets:** The `vod-private` bucket has no public access — all video access requires a valid presigned URL

## License

Internal project — not for public distribution.
