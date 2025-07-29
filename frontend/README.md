# Frontend - Vue.js Web Application

Vue.js web application for My Nethesis with Logto authentication and Role-Based Access Control (RBAC) interface.

## Quick Start

### Prerequisites
- Node.js 20+ LTS
- NPM
- Backend API running
- Logto instance configured

### Setup

```bash
# Install dependencies
npm ci

# Start development server
npm run dev

# Access application at http://localhost:5173
```

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

### Container Development

#### Podman Development
```bash
# Start development container
./dev.sh

# Build container image
./dev.sh build

# Run commands in container
./dev.sh npm run lint-fix
./dev.sh npm run format-fix
./dev.sh bash
```

#### VSCode Dev Containers

**Important Notes:**
- Modifying `dev.containers.dockerPath` setting affects all projects globally
- This procedure may not work on [VSCodium](https://vscodium.com/)

**Setup:**
1. Install [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Configure Podman support:
   - Go to `File > Preferences > Settings`
   - Search for `dev.containers.dockerPath`
   - Set the value to `podman`
3. Open the frontend directory in VSCode
4. Open Command Palette (`CTRL+SHIFT+P`) → "Reopen in Container" (or "Rebuild and Reopen in Container")
5. Open integrated terminal: `View > Terminal`
6. Run development commands:
   ```bash
   npm install          # Install dependencies
   npm run dev          # Start development server
   npm run lint-fix     # Fix linting issues
   npm run format-fix   # Format source files
   npm run qa           # Start QA environment server
   ```

Container configuration is in `.devcontainer/devcontainer.json`.

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
├── .devcontainer/         # VSCode Dev Container config
├── dev.sh                 # Podman development script
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
```

## Related
- [Backend](../backend/README.md) - API server
- [sync CLI](../sync/README.md) - RBAC configuration tool
- [Collect](../collect/README.md) - Collect server
- [Project Overview](../README.md) - Main documentation