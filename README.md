# Pikyon 
### Your Personal Visual Memoir — A Customized Google Photos Experience

Pikyon is a full-stack web application for preserving, organizing, and sharing
personal memories with AI-powered features including song recommendations,
mood analysis, and smart captions.

---

##  Features
-  Upload and organize photos and videos
-  Private lock with PIN protection
-  AI song recommendation based on memory mood
-  Multi-language support
-  Share memories with friends and family
-  Beautiful, warm, cinematic UI

---

##  Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React + TypeScript + Tailwind CSS + Framer Motion |
| Backend API | Go (Gin) |
| AI Service | Python (FastAPI) + Google Gemini |
| Database | PostgreSQL (Supabase) |
| Media Storage | Supabase Storage |
| Deployment | Vercel (frontend) + Railway (backend + AI) |

---

##  Project Structure
pikyon/

├── frontend/       # React + TypeScript app

├── backend/        # Go REST API

├── ai-service/     # Python AI microservice

├── docs/           # Documentation

└── README.md

---

##  Quick Start

### Prerequisites
- Node.js 18+
- Go 1.22+
- Python 3.11+
- Supabase account

### 1. Clone the repo
```bash
git clone https://github.com/ApiyoMargaret/pikyon.git
cd pikyon
```

### 2. Setup backend
```bash
cd backend
cp .env.example .env
# Fill in your environment variables
go mod tidy
go run *.go
```

### 3. Setup AI service
```bash
cd ai-service
pip install -r requirements.txt
uvicorn main:app --reload --port 8001
```

### 4. Setup frontend
```bash
cd frontend
npm install
npm run dev
```

---

##  Supported Languages
English, Swahili, French, Spanish, Arabic

---

##  Documentation
See the [docs](./docs) folder for full API reference and architecture guide.

---

##  Author
Margaret Apiyo — [GitHub](https://github.com/ApiyoMargaret)

##  License
MIT