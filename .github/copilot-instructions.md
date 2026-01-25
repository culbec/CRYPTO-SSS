# CRYPTO-SSS Project Instructions

## Project Overview
Full-stack cryptographic voting application demonstrating **Shamir's Secret Sharing Scheme (SSS)** with hierarchical access structures. Go backend implements SSS from scratch for secure poll results; Angular frontend provides UI (currently minimal).

**Core Domain**: Threshold cryptography for voting systems where results are encrypted and require collaborative decryption by auditors + election officials.

## Architecture

### Backend (Go + MongoDB)
- **Entry**: [cmd/backend.go](../src/backend/cmd/backend.go) - Swagger-documented API on port 3000
- **Core SSS**: [pkg/sss/](../src/backend/pkg/sss/) - Custom Shamir implementation with access structures
  - `sss.go`: (t,n) threshold secret sharing over GF(2^256 - 189)
  - `access_structure.go`: Hierarchical policies (e.g., "1 auditor AND 2 of 3 officials")
- **API Handlers**: [internal/api/](../src/backend/internal/api/) - poll, ballot, auth, admin endpoints
- **Auth**: JWT-based with role middleware (voter/auditor/official/admin) in [auth/middleware.go](../src/backend/internal/api/auth/middleware.go)
- **DB**: MongoDB collections for Users, Polls, EncryptedBallots, SecretShares via [pkg/mongo/](../src/backend/pkg/mongo/)
- **Security**: Argon2id password hashing in [pkg/security/](../src/backend/pkg/security/)

### Frontend (Angular 19)
- Standalone components, minimal routing in [app/app.routes.ts](../src/frontend/src/app/app.routes.ts)
- **Status**: Largely TODO - no feature modules yet

### Data Flow
1. **Poll Creation** → Official creates poll with access structure thresholds (e.g., 1/2 auditors, 2/3 officials)
2. **Voting** → Voters cast encrypted ballots; stored with commitments
3. **Freeze** → Poll closed, ballot commitment hash published
4. **Reveal** → Participants submit their shares; backend reconstructs key via SSS when threshold met

## Development Workflows

### Running Backend
```bash
# With Docker (preferred)
make build-backend  # Builds & starts backend + MongoDB
make run-backend    # Runs existing containers

# Local development with hot reload
cd src/backend
air  # Requires: go install github.com/air-verse/air@latest

# Manual build
cd src/backend
go build -o ./build
./build
```

### Configuration
- [configs/config.json](../src/backend/configs/config.json): DB URI, JWT secret, seeding toggle
- **Seeding**: `"seed_data": true` auto-populates sample users/polls on startup via [internal/seeder/](../src/backend/internal/seeder/)
- **Docker secrets**: HuggingFace token in `deployments/HF_TOKEN` (for future ML features)

### Testing
```bash
cd src/backend
go test ./test/...  # All tests in test/ directory
go test ./test/sss_test.go -v  # Specific test file
```
Tests use table-driven patterns with `t.Run()` subtests. See [test/access_structure_test.go](../src/backend/test/access_structure_test.go) for SSS examples.

### API Documentation
- Swagger annotations in handlers (`@Summary`, `@Description`, `@Tags`)
- Generated docs: [docs/swagger.json](../src/backend/docs/swagger.json)
- Access at `http://localhost:3000/swagger/index.html` when running
- Regenerate: `swag init -g cmd/backend.go -o docs`

## Project-Specific Conventions

### Code Organization
- **Types**: All domain models in [internal/types/types.go](../src/backend/internal/types/types.go) (User, Poll, EncryptedBallot, SecretShare, etc.)
- **Constants**: Centralized in [internal/constants.go](../src/backend/internal/constants.go) (Argon2id params, collection names)
- **Logging**: Structured logging via `slog` with context propagation - use `logging.FromContext(ctx)`

### Naming & Patterns
- **Handlers**: Each API area has `New{Feature}Handler(db)` constructor returning handler struct
- **Middleware**: Auth stored in context key `"username"` via `auth.ContextUsernameKey`
- **Routes**: Grouped by feature in [server/routes.go](../src/backend/internal/server/routes.go) with role guards:
  ```go
  polls := s.router.Group("/api/polls").Use(RequireAuth(authHandler))
  polls.POST("/", RequireRole(types.RoleOfficial), pollHandler.CreatePoll)
  ```
- **MongoDB**: Use `mongo.DbCollections` map for collection names; custom client in [pkg/mongo/mongo.go](../src/backend/pkg/mongo/mongo.go)

### SSS Implementation Details
- **Field**: All arithmetic in GF(p) where p = 2^256 - 189
- **Share Generation**: Polynomial evaluation with Horner's method
- **Reconstruction**: Lagrange interpolation with modular inverse
- **Access Structures**: Tree-based (AND/OR/LEAF nodes) with hierarchical key splitting
  - Example: `AccessStructure.AddGroup("auditors", 1, 2)` then `SetAccessTree(andNode)`

### Security Practices
- JWT secret in config, validated on all protected routes
- Passwords hashed with Argon2id (time=1, memory=64MB, threads=4, keyLen=32)
- Share commitments prevent tampering during reveal phase
- **Never** log sensitive data (passwords, shares, secrets)

## Key Files for Understanding

- [pkg/sss/sss.go](../src/backend/pkg/sss/sss.go): Core SSS split/combine algorithms
- [pkg/sss/access_structure.go](../src/backend/pkg/sss/access_structure.go): Hierarchical secret sharing logic
- [internal/types/types.go](../src/backend/internal/types/types.go): Full domain model with poll lifecycle states
- [internal/api/ballot/ballot.go](../src/backend/internal/api/ballot/ballot.go): Voting & reveal flow
- [test/access_structure_test.go](../src/backend/test/access_structure_test.go): Access structure scenarios

## Common Tasks

### Adding a New API Endpoint
1. Define request/response types in [types/types.go](../src/backend/internal/types/types.go)
2. Implement handler method with Swagger annotations
3. Register route in [server/routes.go](../src/backend/internal/server/routes.go) with appropriate middleware
4. Regenerate Swagger docs: `swag init -g cmd/backend.go -o docs`

### Modifying SSS Logic
- Edit [pkg/sss/](../src/backend/pkg/sss/), add tests in [test/sss_test.go](../src/backend/test/sss_test.go)
- Verify field arithmetic stays in GF(Prime) - use `result.Mod(result, Prime)`
- Access structure changes require validating tree integrity via `validateAccessTree()`

### Frontend Integration (when ready)
- Backend runs on `localhost:3000`, CORS configured in [server/middleware.go](../src/backend/internal/server/middleware.go)
- Auth: Send JWT in `Authorization: Bearer <token>` header
- Poll states: `draft → open → closed → frozen → revealed` (see [types.go](../src/backend/internal/types/types.go))
