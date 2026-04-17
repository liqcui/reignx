# ReignX Web Console

Modern web-based management interface for the ReignX distributed server management platform.

## Features

- **Dashboard**: Real-time server statistics, task metrics, and system health monitoring
- **Server Management**: Browse, search, and manage server inventory
- **Job Management**: Create, monitor, and manage jobs for OS installation, patching, and package deployment
- **Task Monitoring**: Track individual task execution across all servers
- **Web Terminal**: Browser-based SSH terminal for remote server access
- **Responsive Design**: Works on desktop, tablet, and mobile devices

## Technology Stack

- **React 18.2** - UI framework
- **TypeScript** - Type-safe development
- **Vite** - Fast build tool and dev server
- **Ant Design 5.12** - Enterprise UI component library
- **React Router 6.21** - Client-side routing
- **Zustand 4.4.7** - Lightweight state management
- **xterm.js 5.3.0** - Terminal emulator
- **Recharts 2.10.3** - Data visualization
- **Axios 1.6.2** - HTTP client

## Development Setup

### Prerequisites

- Node.js 18+ and npm
- Go 1.22+ (for backend)

### Installation

1. Install dependencies:

```bash
cd web/frontend
npm install
```

2. Start development server:

```bash
npm run dev
```

The frontend will be available at http://localhost:3000

### Development Mode

In development mode, Vite dev server runs on port 3000 and proxies API requests to the backend on port 8080:

- `/api/*` → `http://localhost:8080/api/*` (REST API)
- `/ws/*` → `ws://localhost:8080/ws/*` (WebSocket)

### Build for Production

```bash
npm run build
```

Built files will be in `dist/` directory and will be embedded into the Go binary.

## Project Structure

```
web/frontend/
├── src/
│   ├── components/
│   │   ├── Layout/
│   │   │   └── MainLayout.tsx      # Main application layout
│   │   └── Terminal/
│   │       └── Terminal.tsx        # WebSocket terminal component
│   ├── pages/
│   │   ├── Dashboard.tsx           # Dashboard with metrics
│   │   ├── Servers.tsx             # Server list page
│   │   ├── ServerDetail.tsx        # Server details with terminal
│   │   ├── Jobs.tsx                # Job list page
│   │   ├── JobDetail.tsx           # Job details and progress
│   │   ├── Tasks.tsx               # Task list page
│   │   └── Login.tsx               # Login page
│   ├── stores/
│   │   └── authStore.ts            # Authentication state
│   ├── App.tsx                     # Main application component
│   ├── main.tsx                    # Application entry point
│   └── index.css                   # Global styles
├── index.html                      # HTML template
├── package.json                    # NPM dependencies
├── tsconfig.json                   # TypeScript configuration
├── vite.config.ts                  # Vite configuration
└── README.md                       # This file
```

## Features

### Dashboard

- Server statistics (total, active, failed)
- Pending jobs count
- 24-hour task execution metrics (line chart)
- System health (CPU, memory, disk)
- Recent jobs table

### Server Management

- Server list with search and filtering
- Server details page with:
  - Real-time metrics (CPU, memory, disk)
  - Pending patches count
  - Package inventory
  - Web terminal access
  - Server information

### Job Management

- Create new jobs (patch, package, deploy, upgrade, install_os)
- Monitor job progress
- View task breakdown by status
- Cancel running jobs
- Retry failed jobs

### Task Monitoring

- View all tasks across jobs
- Filter by status, type, server
- Task execution details
- Retry failed tasks

### Web Terminal

- Browser-based SSH terminal using xterm.js
- Real-time terminal emulation
- WebSocket-based communication
- Automatic reconnection
- Terminal resize support

## API Integration

The frontend communicates with the backend via:

1. **REST API** (`/api/v1/*`):
   - Authentication: `POST /api/v1/auth/login`
   - Servers: `GET /api/v1/servers`, `GET /api/v1/servers/:id`
   - Jobs: `GET /api/v1/jobs`, `POST /api/v1/jobs`, `GET /api/v1/jobs/:id`
   - Tasks: `GET /api/v1/tasks`, `GET /api/v1/tasks/:id`
   - Metrics: `GET /api/v1/metrics/dashboard`

2. **WebSocket** (`/ws/*`):
   - Terminal: `ws://host/ws/terminal/:serverID`

## Authentication

The web console uses JWT-based authentication:

1. User logs in with username/password
2. Backend returns access token and refresh token
3. Access token is stored in Zustand state (persisted to localStorage)
4. All API requests include `Authorization: Bearer <token>` header
5. Access tokens expire after 1 hour
6. Refresh tokens can be used to obtain new access tokens

Default credentials (development):
- Username: `admin`
- Password: `admin`

## Configuration

### Environment Variables

Create `.env.local` for local development:

```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_BASE_URL=ws://localhost:8080
```

### Vite Proxy Configuration

The `vite.config.ts` file configures the development proxy:

```typescript
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
    '/ws': {
      target: 'ws://localhost:8080',
      ws: true,
    },
  },
}
```

## Terminal Component

The terminal component uses xterm.js for terminal emulation:

```typescript
import Terminal from '@/components/Terminal/Terminal'

<Terminal serverId="server-001" />
```

Features:
- Full terminal emulation (colors, cursor control)
- WebSocket-based communication
- Automatic resize on window resize
- Copy/paste support
- Link detection (web-links addon)

## State Management

Authentication state is managed with Zustand:

```typescript
import { useAuthStore } from '@/stores/authStore'

const { user, token, isAuthenticated, login, logout } = useAuthStore()
```

The state is automatically persisted to localStorage.

## Styling

- Ant Design theme customization in `App.tsx`
- Global styles in `index.css`
- Component-specific styles using inline styles or CSS modules

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

## Security

- JWT tokens stored in localStorage (auto-cleared on logout)
- CORS headers configured on backend
- WebSocket connections use same origin policy
- Input validation on all forms
- XSS protection via React's auto-escaping

## TODO

- [ ] Add real-time updates with Server-Sent Events or WebSocket
- [ ] Implement refresh token rotation
- [ ] Add user management UI
- [ ] Add audit log viewer
- [ ] Add package inventory details
- [ ] Add patch history viewer
- [ ] Add server grouping/tagging
- [ ] Add dark mode support
- [ ] Add internationalization (i18n)
- [ ] Add unit tests with Vitest
- [ ] Add E2E tests with Playwright
