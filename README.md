# Kredly

Kredly adalah platform penilaian dan verifikasi kredensial berbasis AI yang menggabungkan adaptive testing, sertifikasi berbasis blockchain, dan analisis CV cerdas untuk memberikan validasi keterampilan dan alat pengembangan karir yang komprehensif.

## Fitur

### 🎯 Computerized Adaptive Testing (CAT)

- **Penilaian berbasis IRT**: Mengimplementasikan Item Response Theory untuk pengujian keterampilan yang personal dan adaptif
- **Tingkat Kesulitan Dinamis**: Pertanyaan secara otomatis menyesuaikan berdasarkan performa pengguna
- **Evaluasi Komprehensif**: Penilaian multi-topik dengan feedback detail, kekuatan, dan rekomendasi
- **Sesi yang Dapat Dilanjutkan**: Persistensi sesi 24 jam memungkinkan pengguna melanjutkan penilaian yang terputus

### 📄 Analisis CV Berbasis AI

- **Parsing Cerdas**: Ekstrak dan analisis konten CV menggunakan Groq AI
- **Penilaian Keterampilan**: Buat penilaian khusus berdasarkan konten CV
- **Wawasan Karir**: Dapatkan rekomendasi personal dan saran perbaikan

### 🔐 Verifikasi Sertifikat Blockchain

- **Integrasi Ethereum**: Terbitkan sertifikat anti-rusak di blockchain
- **Penyimpanan IPFS**: Penyimpanan sertifikat terdesentralisasi via Pinata
- **Verifikasi Publik**: Siapa saja dapat memverifikasi keaslian sertifikat menggunakan ID sertifikat atau hash PDF
- **Catatan Permanen**: Catatan pencapaian yang permanen dan dapat diverifikasi

### 👤 Autentikasi & Profil Pengguna

- **Berbagai Metode Autentikasi**: Google OAuth dan Email OTP authentication
- **Profil Publik**: Halaman profil publik yang dapat disesuaikan dengan kontrol privasi
- **Foto Profil**: Penyimpanan foto profil berbasis IPFS
- **Sistem Username**: Username unik untuk URL profil publik

### 💼 Pencocokan Pekerjaan

- **Pencocokan Berbasis AI**: Rekomendasi pekerjaan cerdas berdasarkan keterampilan dan penilaian
- **Pelacakan Pekerjaan**: Simpan dan kelola peluang pekerjaan
- **Log Aktivitas**: Lacak semua aktivitas penilaian dan sertifikasi

### 💬 Asisten Chat AI

- **Didukung Groq**: AI percakapan untuk panduan karir dan dukungan platform
- **Context-aware**: Memahami profil pengguna dan riwayat penilaian

### 🪙 Sistem Token

- **Berbasis Kredit**: Sistem token untuk fitur premium dan penilaian
- **Manajemen Saldo**: Lacak dan isi ulang saldo token

## Tech Stack

### Frontend

- **Framework**: React 19 dengan Rsbuild
- **Routing**: TanStack Router dengan auto code-splitting
- **Styling**: Tailwind CSS 4 + komponen shadcn/ui
- **State Management**: Zustand
- **Animations**: Motion (Framer Motion)
- **UI Components**: Radix UI primitives, Lucide icons
- **Build Tool**: Rsbuild dengan React Compiler, image compression, dan SVGR

### Backend

- **Language**: Go 1.26.1
- **Framework**: Gin (HTTP router)
- **Database**: MongoDB
- **AI/ML**: Groq API untuk fitur berbasis LLM
- **Blockchain**: Ethereum (go-ethereum)
- **Storage**: IPFS via Pinata
- **Email**: Resend API
- **Authentication**: OAuth2 (Google), Email OTP

### Development Tools

- **Hot Reload**: Air (Go), Rsbuild dev server (React)
- **Type Checking**: TypeScript 6
- **Linting**: ESLint dengan React hooks rules
- **Formatting**: Prettier
- **Package Manager**: pnpm

## Struktur Proyek

```
kredly/
├── src/                    # Source frontend
│   ├── components/         # Komponen React reusable
│   ├── pages/             # Komponen halaman
│   │   ├── client/        # Halaman publik (landing, features, about)
│   │   └── dashboard/     # Halaman dashboard terproteksi
│   ├── contexts/          # React contexts (auth, dll)
│   ├── hooks/             # Custom React hooks
│   ├── lib/               # Utilities dan helpers
│   ├── routes/            # TanStack Router routes
│   ├── services/          # API client services
│   └── stores/            # Zustand stores
├── server/                # Source backend
│   ├── cmd/api/          # Entry point aplikasi
│   ├── internal/         # Package internal
│   │   ├── blockchain/   # Integrasi Ethereum & IPFS
│   │   ├── config/       # Manajemen konfigurasi
│   │   ├── database/     # Koneksi MongoDB
│   │   ├── groq/         # Groq AI client
│   │   ├── handlers/     # HTTP request handlers
│   │   ├── middleware/   # HTTP middleware (auth, CORS)
│   │   ├── models/       # Model data
│   │   ├── pdf/          # Generasi & parsing PDF
│   │   ├── service/      # Business logic services
│   │   └── store/        # Data access layer
│   ├── uploads/          # File yang diupload user
│   └── .air.toml         # Konfigurasi Air hot reload
├── public/               # Aset statis
└── dist/                 # Output production build
```

## Memulai

### Prasyarat

- **Node.js**: v18+
- **Go**: 1.26.1+
- **pnpm**: Versi terbaru
- **MongoDB**: Instance yang berjalan
- **Air**: Go hot reload tool (`go install github.com/air-verse/air@latest`)

### Environment Variables

#### Konfigurasi Backend

Buat `server/.env` berdasarkan `server/.env.example`:

```bash
# Database
DATABASE_URL=mongodb://localhost:27017/kredly

# Server
PORT=8080
ENVIRONMENT=development
API_URL=http://localhost:8080
FRONTEND_URL=http://localhost:3000

# Authentication
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret

# Email Service
RESEND_API_KEY=your_resend_api_key

# AI/LLM
GROQ_API_KEY=your_groq_api_key
GROQ_BASE_URL=https://api.groq.com/openai/v1

# Job Scraping
APIFY_TOKEN=your_apify_token

# Blockchain
BLOCKCHAIN_RPC_URL=your_ethereum_rpc_url
BLOCKCHAIN_CONTRACT_ADDRESS=your_contract_address
BLOCKCHAIN_PRIVATE_KEY=your_private_key

# IPFS Storage
PINATA_JWT=your_pinata_jwt_token
```

### Instalasi

1. **Clone repository**

```bash
git clone <repository-url>
cd kredly
```

2. **Install dependensi frontend**

```bash
pnpm install
```

3. **Install dependensi backend**

```bash
cd server
go mod download
```

### Development

#### Menjalankan Backend Server

```bash
# Dari root proyek
pnpm run dev:server

# Atau manual dari direktori server
cd server
air
```

Backend berjalan di `http://localhost:8080`

#### Menjalankan Frontend Dev Server

```bash
# Dari root proyek
pnpm run dev
```

Frontend berjalan di `http://localhost:3000`

### Build untuk Production

#### Build Frontend

```bash
pnpm run build
```

Output: direktori `dist/`

#### Build Backend

```bash
pnpm run build:server
```

Output: binary `server/tmp/main`

### Code Quality

#### Menjalankan Type Checking

```bash
pnpm run typecheck
```

#### Menjalankan Linter

```bash
pnpm run lint
```

#### Format Code

```bash
pnpm run format
```

## Dokumentasi API

### Endpoint Authentication

- `GET /api/auth/sign-in/google` - Memulai alur Google OAuth
- `GET /api/auth/callback/google` - Callback Google OAuth
- `POST /api/auth/sign-in/email-otp` - Request email OTP
- `POST /api/auth/verify/email-otp` - Verifikasi OTP dan login
- `GET /api/auth/get-session` - Dapatkan sesi pengguna saat ini (terproteksi)
- `POST /api/auth/sign-out` - Logout

### Endpoint Assessment (Terproteksi)

- `POST /api/sessions` - Buat sesi CAT baru
- `GET /api/sessions/:id` - Dapatkan detail sesi
- `GET /api/sessions/:id/next-item` - Dapatkan pertanyaan adaptif berikutnya
- `POST /api/sessions/:id/answer` - Submit jawaban
- `GET /api/sessions/:id/result` - Dapatkan hasil penilaian akhir
- `POST /api/sessions/:id/abandon` - Tinggalkan sesi

### Endpoint Pemrosesan CV (Terproteksi)

- `POST /api/parse-cv` - Parse dan analisis CV
- `POST /api/profile/custom-assessment` - Generate penilaian khusus dari CV

### Endpoint Blockchain

- `POST /api/blockchain/issue` - Terbitkan sertifikat di blockchain (terproteksi)
- `GET /api/blockchain/verify` - Verifikasi sertifikat berdasarkan ID
- `POST /api/blockchain/verify-by-hash` - Verifikasi sertifikat berdasarkan hash PDF
- `GET /api/certificates/metadata/:sessionId` - Dapatkan metadata sertifikat berdasarkan sesi
- `GET /api/certificates/metadata/cert/:certificateId` - Dapatkan metadata sertifikat berdasarkan ID sertifikat

### Endpoint User (Terproteksi)

- `GET /api/user/me/token-balance` - Dapatkan saldo token pengguna
- `POST /api/user/me/topup` - Isi ulang token (simulasi)
- `PUT /api/user/update-profile` - Update profil pengguna
- `POST /api/user/upload-cv` - Upload file CV
- `POST /api/user/upload-profile-photo` - Upload foto profil ke IPFS
- `DELETE /api/user/delete-account` - Hapus akun pengguna
- `GET /api/user/public-profile-settings` - Dapatkan pengaturan profil publik
- `PUT /api/user/public-profile-settings` - Update pengaturan profil publik

### Endpoint Profile

- `GET /api/check-username` - Cek ketersediaan username (publik)
- `GET /api/profile` - Dapatkan profil sendiri (terproteksi)
- `GET /api/profile/public/:username` - Dapatkan profil publik berdasarkan username

### Endpoint Job (Terproteksi)

- `POST /api/jobs/fetch` - Ambil dan simpan pekerjaan
- `GET /api/jobs` - Dapatkan pekerjaan yang disimpan pengguna

### Endpoint Lainnya

- `GET /api/health` - Health check
- `GET /api/activities` - Dapatkan aktivitas pengguna (terproteksi)
- `POST /api/chat` - Chat dengan asisten AI

## Teknologi & Pattern Utama

### Sistem CAT (Computerized Adaptive Testing)

- **Item Response Theory (IRT)**: Menggunakan estimasi theta (kemampuan) dan kesulitan item (b-parameter)
- **Pemilihan Pertanyaan Adaptif**: Memilih pertanyaan yang sesuai dengan tingkat kemampuan pengguna saat ini
- **Standard Error Measurement (SEM)**: Menghentikan penilaian ketika ambang presisi tercapai
- **Min/Max Items**: Jumlah pertanyaan minimum dan maksimum yang dapat dikonfigurasi
- **Persistensi Sesi**: TTL 24 jam dengan sliding window untuk sesi yang dapat dilanjutkan

### Alur Authentication

- **Kompatibel Better Auth**: Endpoint mengikuti konvensi API Better Auth
- **JWT Tokens**: Autentikasi berbasis token yang aman
- **Auto-refresh**: Middleware menangani refresh token secara otomatis
- **Berbasis Cookie**: Menggunakan HTTP-only cookies untuk penyimpanan token yang aman

### Integrasi Blockchain

- **Penerbitan Sertifikat**: Generate PDF, hash, simpan di IPFS, catat hash di blockchain
- **Verifikasi Ganda**: Verifikasi berdasarkan ID sertifikat atau hash file PDF
- **Penyimpanan Metadata**: MongoDB menyimpan metadata sertifikat untuk lookup cepat
- **Pinata IPFS**: Penyimpanan terdesentralisasi permanen untuk sertifikat

## Deployment

### Dukungan Vercel

Aplikasi secara otomatis mendeteksi deployment Vercel dan menyesuaikan base path API.

### Deteksi Environment

- Development: Menggunakan base path `/api` dengan proxy
- Production/Vercel: Dapat dikonfigurasi berdasarkan environment variables

## Kontribusi

1. Fork repository
2. Buat feature branch (`git checkout -b feature/fitur-keren`)
3. Commit perubahan Anda (`git commit -m 'feat: tambah fitur keren'`)
4. Push ke branch (`git push origin feature/fitur-keren`)
5. Buka Pull Request

### Konvensi Commit

Ikuti format conventional commits:

- `feat:` - Fitur baru
- `fix:` - Perbaikan bug
- `refactor:` - Refaktor kode
- `docs:` - Perubahan dokumentasi
- `style:` - Perubahan gaya kode
- `test:` - Penambahan/perubahan test
- `chore:` - Perubahan proses build/tooling

## Lisensi

[Tambahkan informasi lisensi]

## Dukungan

Untuk masalah, pertanyaan, atau kontribusi, silakan buka issue di GitHub.
