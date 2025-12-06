# 🔴 Connect 4 (Real-Time Multiplayer Game)

A production-ready, distributed multiplayer game built with **Golang**, **React**, and **Event-Driven Architecture**.

It features real-time WebSocket communication, a competitive Minimax Bot, PostgreSQL persistence, and a Kafka-based analytics pipeline.

## 🚀 Live Demo

| Component | URL | Status |
|-----------|-----|--------|
| **Play the Game** | [https://connectfour-zeta.vercel.app](https://connectfour-zeta.vercel.app) | 🟢 Live |
| **Backend API** | [https://connect4-backend-ov70.onrender.com/health](https://connect4-backend-ov70.onrender.com/health) | 🟢 Live |
| **Leaderboard** | [https://connectfour-zeta.vercel.app](https://connectfour-zeta.vercel.app) (Click "View Leaderboard") | 🟢 Live |

> **⚠️ Deployment Architecture:** The live backend runs on a Free Tier (Render). We utilize a **Graceful Degradation** strategy for the Analytics service:
>
> *   **Live Environment:** The application detects that no Kafka provider is configured. It automatically disables the analytics module but keeps the Game Engine and Database fully operational.
> *   **Local Environment:** The application connects to the local Dockerized Kafka instance, enabling the full real-time analytics pipeline.

## 🎮 Features

*   **Real-Time Gameplay:** Instant state synchronization using WebSockets.
*   **Smart Matchmaking:** Pairs players automatically. If no opponent is found in 10s, a Bot joins.
*   **Rejoin Capability:** If a player disconnects, they can rejoin the active game within 30 seconds.
*   **Forfeit Logic:** If a disconnected player doesn't return in 30s, the game is forfeited.
*   **Persistence:** Every game result is stored in PostgreSQL.
*   **Analytics Pipeline:** Game events (duration, winner) are streamed to Apache Kafka and consumed by an internal analytics service to track metrics.
*   **Leaderboard:** Displays top players based on wins.

## 🛠️ Tech Stack

*   **Backend:** Golang 1.24 (Gorilla WebSocket, IBM Sarama, Lib/PQ)
*   **Frontend:** React + Vite
*   **Database:** PostgreSQL 15 (Supabase for Prod, Docker for Local)
*   **Messaging:** Apache Kafka + Zookeeper (Docker for Local, Gracefully Disabled in Prod)
*   **Infrastructure:** Docker Compose, Render (Backend), Vercel (Frontend)

## ⚙️ Local Setup Guide

Follow these detailed steps to run the full stack (including Kafka) locally.

### Prerequisites

Ensure you have the following installed:
*   **Docker Desktop** (Essential for DB & Kafka)
*   **Go 1.21+**
*   **Node.js 18+**

### 1. Start Infrastructure (DB & Messaging)

We use Docker Compose to spin up isolated containers for Postgres, Zookeeper, and Kafka.

Open a terminal in the root `CONNECTFOUR` folder.

Run the following command to download images and start containers:

```bash
docker compose up -d
```

**Verify:** Check that all three containers are "Up" or "Running":

```bash
docker ps
```

*(You should see `connect4_db`, `connect4_kafka`, and `connect4_zookeeper`)*.

### 2. Start Backend Server

The backend handles the game logic, WebSocket connections, and database interactions.

Navigate to the backend directory:

```bash
cd backend
```

Install Go dependencies:

```bash
go mod tidy
```

**Set Environment Variable:** You must tell the app to attempt a local Kafka connection.

*Mac / Linux / Git Bash:*

```bash
export KAFKA_ENABLE_LOCAL=true
```

*Windows PowerShell:*

```powershell
$env:KAFKA_ENABLE_LOCAL="true"
```

Run the server:

```bash
go run cmd/server/main.go
```

**Success:** You will see `Connected to Postgres successfully`, `Local Kafka Connected`, and `Server running on http://0.0.0.0:8080`.

### 3. Start Frontend Client

Open a new terminal window (leave the backend running).

Navigate to the frontend directory:

```bash
cd frontend
```

Install React dependencies:

```bash
npm install
```

Start the development server:

```bash
npm run dev
```

Open your browser to the URL shown (usually `http://localhost:5173`).

## 🧪 How to Verify Features

### Bot Match
1.  Open the frontend. Enter "Alice" and join.
2.  Wait 10 seconds. The "Bot" will join automatically.
3.  Play to win.

### PVP Match
1.  Open two browser tabs (e.g., one Incognito).
2.  Tab 1: Join as "P1". Tab 2: Join as "P2".
3.  Match starts instantly.

### Analytics Verification (Local Only)
1.  Finish a game locally.
2.  Check your Backend Terminal logs. You will see a real-time report:
    ```text
    📊 ANALYTICS PROCESSED: Winner=Alice GameID=... Duration=42.5s
    ```

### Rejoin Test
1.  In an active game, close the tab or refresh.
2.  Reopen and join with the exact same username within 30s.
3.  You will be reconnected to the board state exactly where you left off.

## 📂 Project Structure

```text
CONNECTFOUR/
├── docker-compose.yml       # Orchestration for Local Kafka & Postgres
├── backend/
│   ├── cmd/server/main.go   # Application Entry Point & Config
│   ├── internal/
│   │   ├── api/             # WebSocket & HTTP Route Handlers
│   │   ├── game/            # Core Game Engine (Rules, Minimax Bot, Lobby Hub)
│   │   ├── db/              # Postgres Repository Implementation
│   │   └── event/           # Kafka Producer & Consumer Logic
│   └── go.mod
└── frontend/
    ├── src/
    │   └── App.jsx          # Full React Frontend (UI, WebSocket, State)
    └── package.json
```
