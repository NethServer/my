# My Nethesis Frontend

This project is a TypeScript-based [Vue.js](https://vuejs.org/) application built
using [Vite](https://vitejs.dev/).

### Environment

My Nethesis frontend requires an environment file to communicate with the backend API and Logto. This file should be named `.env.<environment>` (e.g., `.env.development` or `.env.qa`) and must be generated using the `sync` command. For more information, please refer to the [sync README](https://github.com/NethServer/my/blob/main/sync/README.md).

### Commit notations

To maintain clear and linear commit history, the project adheres to the [Conventional Commits Specification v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

### Code style

The codebase follows a consistent style enforced by [Prettier](https://prettier.io) in combination with [ESLint](https://eslint.org). When opening a pull request, it is mandatory to check for linting errors and ensure the code is properly formatted. This helps maintain consistency and prevents issues during code review.

To format all source files use the NPM script `format-fix`, while for checking for (and possibly fixing) linting errors use the `lint-fix` script. Refer to the sections below for instructions on how to run these scripts.

### Development in a container

You may choose between two development environments:

- [Podman development container](#podman-development-container)
- [VSCode Dev Containers](#use-vscode-dev-containers)

#### Podman development container

This option allows you to run the container independently of any specific IDE or editor.

To start the development container, simply run:

```bash
./dev.sh
```

This command builds the required image, installs Node.js dependencies, and starts a development server at `http://localhost:5173`.

The `dev.sh` script can also perform additional tasks. For example, to rebuild the image:

```bash
./dev.sh build
```

Or to execute commands directly inside the container:

```bash
# Check for (and possibly fix) linting errors
./dev.sh npm run lint-fix

# Format all source files
./dev.sh npm run format-fix

# Start the development server using the QA (quality assurance) environment
./dev.sh npm run qa

# Add a new NPM package to the project
./dev.sh npm install cool-package

# Access the container shell
./dev.sh bash
```

#### VSCode Dev Containers

Please note that:
- This procedure involves modifying the `dev.containers.dockerPath` setting, which is global. This may impact other projects using VSCode and Dev Containers
- This procedure may not work on [VSCodium](https://vscodium.com/)

To develop My Nethesis frontend using VSCode Dev Containers:

- Install VSCode
  extension [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
- By default, Dev Containers uses Docker, but it can be configured to use Podman:
  - Go to `File > Preferences > Settings`
  - Search for `dev.containers.dockerPath`
  - Set the value to `podman`
- Open the `my-nethesis-ui` directory (the repository root) in VSCode, if you haven't already
- Open the Command Palette (`CTRL+SHIFT+P`) and type `Reopen in Container` (or `Rebuild and Reopen in Container`, if needed)
- Open the integrated terminal: `View > Terminal`
- Enter one of the following commands:
  - `npm install`: install dependencies
  - `npm run dev`: start the development server
  - `npm run lint-fix`: check for (and possibly fix) linting errors
  - `npm run format-fix`: format all source files
  - `npm run qa`: start the development server using the QA (quality assurance) environment

Container configuration is contained inside `.devcontainer/devcontainer.json`.

### Development on your workstation

While container-based development is recommended, you may also work directly on your local system:

- Install Node.js (LTS version) and NPM
- Run a local web server on your workstation:
  - `npm install`: install dependencies
  - `npm run dev`: start the development server
  - `npm run lint-fix`: check for (and possibly fix) linting errors
  - `npm run format-fix`: format all source files
  - `npm run qa`: start the development server using the QA (quality assurance) environment

