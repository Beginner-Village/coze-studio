# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Coze Studio is an all-in-one AI agent development platform with both frontend (React + TypeScript) and backend (Go) components. The project uses a sophisticated monorepo architecture managed by Rush.js with 135+ frontend packages organized in a hierarchical dependency system.

## Development Commands

### Environment Setup
```bash
# Clone and setup
git clone https://github.com/coze-dev/coze-studio.git
cd coze-studio

# Install frontend dependencies
rush update

# For Docker-based development
cd docker
cp .env.example .env
# Configure model settings in backend/conf/model/
docker compose up -d
# Access at http://localhost:8888
```

### Development Workflow
```bash
# Start middleware services (MySQL, Redis, Elasticsearch, etc.)
make middleware

# Start Go backend in development mode
make server

# Start frontend development server
cd frontend/apps/coze-studio
npm run dev

# Full development environment
make debug
```

### Build Commands
```bash
# Build frontend only
make fe

# Build Go server
make build_server

# Build everything with Docker
make web

# Rush monorepo commands
rush build                    # Build all packages
rush rebuild -o @coze-studio/app  # Build specific package
rush test                     # Run all tests
rush lint                     # Lint all packages
```

### Testing
```bash
# Run tests (Vitest-based)
rush test
npm run test                  # In specific package
npm run test:cov             # With coverage

# Backend tests
cd backend && go test ./...
```

## Architecture Overview

### Frontend Architecture
- **Monorepo**: Rush.js with 135+ packages across 4 dependency levels
- **Build System**: Rsbuild (Rspack-based) for fast builds
- **UI Framework**: React 18 + TypeScript + Semi Design + Tailwind CSS
- **State Management**: Zustand for global state
- **Package Organization**:
  - `arch/`: Core infrastructure (level-1)
  - `common/`: Shared components and utilities (level-2)
  - `agent-ide/`, `workflow/`, `studio/`: Feature domains (level-3)
  - `apps/coze-studio`: Main application (level-4)

### Backend Architecture (Go)
- **Framework**: Hertz HTTP framework
- **Architecture**: Domain-Driven Design (DDD) with microservices
- **Structure**:
  - `domain/`: Business logic and entities
  - `application/`: Application services and use cases
  - `api/`: HTTP handlers and routing
  - `infra/`: Infrastructure implementations
  - `crossdomain/`: Cross-cutting concerns

### Key Architectural Patterns
- **Adapter Pattern**: Extensive use for loose coupling between layers
- **Interface Segregation**: Clear contracts between domains
- **Event-Driven**: NSQ message queue for async communication
- **API-First**: Comprehensive OpenAPI specifications

## Database & Infrastructure

### Docker Services Stack
- **Database**: MySQL 8.4.5
- **Cache**: Redis 8.0
- **Search**: Elasticsearch 8.18.0 with SmartCN analyzer
- **Vector DB**: Milvus v2.5.10 for embeddings
- **Storage**: MinIO for object storage
- **Message Queue**: NSQ (nsqlookupd, nsqd, nsqadmin)
- **Configuration**: etcd 3.5

### Database Management
```bash
# Sync database schema
make sync_db

# Dump database schema
make dump_db

# Initialize SQL data
make sql_init

# Atlas migration management
make atlas-hash
```

## Key Development Patterns

### Frontend Package Development
- Each package follows consistent structure with `README.md`, `package.json`, `tsconfig.json`, `eslint.config.js`
- Adapter pattern extensively used for decoupling (e.g., `-adapter` suffix packages)
- Base/Core pattern for shared functionality (e.g., `-base` suffix packages)
- Use workspace references (`workspace:*`) for internal dependencies

### Backend Development
- Follow DDD principles with clear domain boundaries
- Use dependency injection via interfaces
- Implement proper error handling with custom error types
- Write comprehensive tests for domain logic

### Model Configuration
Before deployment, configure AI models in `backend/conf/model/`:
1. Copy template from `backend/conf/model/template/`
2. Set `id`, `meta.conn_config.api_key`, and `meta.conn_config.model`
3. Supported providers: OpenAI, Volcengine Ark, Claude, Gemini, Qwen, DeepSeek, Ollama

For batch model import, use the Python model importer:
```bash
# Import models from YAML configs to database
cd scripts
python model_importer.py --config importer_config.yaml --model-dir ../backend/conf/model/
```

## Testing Strategy

### Coverage Requirements by Package Level
- **Level 1**: 80% coverage, 90% increment
- **Level 2**: 30% coverage, 60% increment
- **Level 3-4**: 0% coverage (flexible)

### Testing Framework
- **Frontend**: Vitest for unit/integration tests
- **Backend**: Go's built-in testing framework
- **E2E**: Separate e2e subspace configuration

## Common Issues & Solutions

### Frontend Development
- Use `rush update` instead of `npm install` at root level
- Build packages in dependency order using `rush build`
- For hot reload issues, check Rsbuild configuration in specific package

### Backend Development
- Ensure middleware services are running (`make middleware`)
- Check database connectivity and schema sync
- Verify model configurations are properly set

### Docker Issues
- Ensure sufficient resources (minimum 2 Core, 4GB RAM)
- Check port conflicts (8888 for frontend, various for services)
- Use `make clean` to reset Docker volumes if needed

## IDL and Code Generation

The project uses Interface Definition Language (IDL) for API contract management:
- IDL files in `idl/` directory (Thrift format)
- Frontend code generation via `@coze-arch/idl2ts-*` packages
- Backend uses generated Go structs

## Plugin Development

For custom plugin development:
- Reference templates in `backend/conf/plugin/pluginproduct/`
- Follow OAuth schema in `backend/conf/plugin/common/oauth_schema.json`
- Configure authentication keys for third-party services

## Contributing

- Use conventional commits via `rush commit`
- Run linting with `rush lint-staged` (pre-commit hook)
- Ensure tests pass before submitting PRs
- Follow team-based package organization and tagging conventions

## System Requirements

### Minimum Hardware Requirements
- 2 Core CPU
- 4 GB RAM
- Docker and Docker Compose pre-installed

### Required Software
- Go >= 1.23.4
- Node.js >= 21 (as specified in rush.json)
- pnpm 8.15.8
- Docker and Docker Compose

## Team Organization and Package Tags

The frontend monorepo uses a team-based organization system with specific tags:

### Team Tags
- `team-arch`: Core architecture and infrastructure
- `team-automation`: Build and automation tools
- `team-builder`: Build system components
- `team-community`: Community-facing features
- `team-data`: Data management and analytics
- `team-devops`: DevOps and deployment tools
- `team-studio`: Studio UI and user-facing features
- `team-qa`: Quality assurance tools
- `team-fullcode-app`: Full-code application components

### Dependency Level Tags
- `level-1`: Core infrastructure packages (highest priority)
- `level-2`: Shared components and utilities
- `level-3`: Feature domain packages
- `level-4`: Application layer packages

### Additional Tags
- `enabled-bundle-diff`: Packages with bundle size monitoring
- `phase-prebuild`: Packages that must be built before others
- `channel-coze`: Coze-specific channel packages
- `rush-tools`: Rush.js tooling packages
- `core`: Core system packages

## Environment Files

The project uses different environment configurations:
- `.env.debug`: Debug environment configuration
- `.env.debug.example`: Example debug environment template
- `.env.example`: Production environment template

Always copy the appropriate example file before starting development.

## Additional Makefile Commands

Beyond the core commands, the Makefile includes:
- `make setup_es_index`: Setup Elasticsearch indices
- `make build_docker`: Build Docker images
- `make down_web`: Stop web server containers
- `make clean`: Reset Docker volumes and clean state
- `make help`: Display all available commands

## Model Management

The project includes a comprehensive model management system (`modelmgr`):
- Database-backed model configuration and storage
- Support for space-specific model configurations
- Model lifecycle management through application services
- Integration with various AI model providers through standardized interfaces

## Acknowledgments

Key dependencies and frameworks:
- **Eino**: Agent and workflow runtime engines, model abstractions
- **FlowGram**: Workflow building engine for frontend canvas editor
- **Hertz**: High-performance Go HTTP framework for microservices