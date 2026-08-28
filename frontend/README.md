# Frontend - Vue.js Web Application

Vue.js web application for My Nethesis with Logto authentication and Role-Based Access Control (RBAC) interface.

## Quick Start

### Prerequisites
- Node.js — the exact version in [`.nvmrc`](../.nvmrc) at the repo root. See [Node version](#node-version).
- Backend API running on port 8080
- Logto instance configured

### Setup

> **Note:** The frontend requires the backend API running on port 8080.
> Start it first with `cd backend && make dev-up && make run`.

```bash
nvm install    # no argument: reads ../.nvmrc, installs that version and switches to it
node -v        # must match ../.nvmrc
npm ci         # npm ships inside Node — nothing separate to install
npm run dev    # development server on port 5173
```

### Node version

One version is pinned for the whole repo, at the **root** rather than here, because
nvm and fnm both search *upward* from the current directory — so a single pin
covers `frontend/` and `docs/` alike.

`.nvmrc` is the only version file, because every manager in practical use here
reads it: both nvm and fnm do.

- **nvm**: `nvm install`, then `nvm use` in every new shell. nvm does **not** switch
  automatically; add the `cd` hook from nvm's "Deeper Shell Integration" section to
  your shell rc if you want it to.
- **fnm**: `fnm install && fnm use`, or `eval "$(fnm env --use-on-cd)"` to automate it.

`engines` in `package.json` is the backstop. On the wrong Node, `npm ci` still
succeeds but prints `EBADENGINE` — treat that as an error, because `npm run
type-check` would then check against `@types/node` for a runtime you are not
actually running.

### Required Environment Variables

Environment files must be generated using the `sync` tool. See [sync README](../sync/README.md) for details.

**Environment Files:**
- `.env.development` - Development environment
- `.env.qa` - QA/Testing environment  
- `.env.example` - Template file with all required variables

```bash
# Example .env.development
VITE_API_BASE_URL=http://localhost:8080
VITE_LOGTO_ENDPOINT=https://your-tenant.logto.app
VITE_LOGTO_APP_ID=your-spa-app-id
VITE_LOGTO_RESOURCES=https://your-domain.com/api
VITE_SIGNIN_REDIRECT_URI=login-redirect
VITE_SIGNOUT_REDIRECT_URI=login
```

## Architecture

### Vue 3 Composition API
- **TypeScript**: Full type safety with Vue TSC
- **Vite**: Fast development and build tooling
- **Vue Router**: Client-side routing with authentication guards
- **Pinia**: State management for auth and app state

### Authentication Flow
- **Logto SDK**: OAuth2/OIDC integration with PKCE
- **JWT Tokens**: Secure token exchange with backend
- **Route Guards**: Protected routes with role-based access
- **Auto-refresh**: Automatic token renewal

### UI Components
- **Nethesis Components**: Custom component library
- **Tailwind CSS**: Utility-first styling
- **FontAwesome**: Icon system
- **Dark Mode**: Theme switching support

## Development

### Basic Commands
```bash
# Run all quality checks (recommended)
npm run pre-commit

# Individual commands
npm run format        # Check code formatting
npm run format-fix    # Fix code formatting
npm run lint          # Run linting
npm run lint-fix      # Fix linting issues
npm run type-check    # TypeScript type checking
npm run test          # Run tests
npm run build         # Build for production
```

### Development Servers
```bash
# Development server
npm run dev

# QA environment server
npm run qa

# Preview production build
npm run preview
```

## Testing

### Manual Testing
```bash
# Run test suite
npm run test

# Coverage report
npm run test -- --coverage
```

### Authentication Testing
1. Start backend server: `cd ../backend && make run`
2. Access frontend: http://localhost:5173
3. Login with Logto credentials
4. Verify RBAC permissions in UI

## Project Structure

```
frontend/
├── src/
│   ├── components/         # Vue components
│   │   ├── account/       # User account management
│   │   ├── customers/     # Customer management
│   │   ├── distributors/  # Distributor management
│   │   ├── resellers/     # Reseller management
│   │   └── users/         # User management
│   ├── lib/               # Utilities and API clients
│   ├── router/            # Vue Router configuration
│   ├── stores/            # Pinia state management
│   ├── views/             # Page components
│   └── i18n/              # Internationalization
├── public/                # Static assets
└── build.sh               # Production build script
```

## Build and Deployment

### Local Build
```bash
# Production build
npm run build

# Build output in dist/
ls dist/
```

### Container Build
```bash
# Production container
./build.sh

# Verify build
podman run -p 8080:80 my-nethesis-frontend:latest
# OR
docker run -p 8080:80 my-nethesis-frontend:latest
```

## Related
- [Backend](../backend/README.md) - API server
- [sync CLI](../sync/README.md) - RBAC configuration tool
- [Collect](../collect/README.md) - Collect server
- [Project Overview](../README.md) - Main documentation