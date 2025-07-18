# 01 - Project Setup

## Overview

Initialize the GoCommender project with proper Go structure following llm-shared guidelines.

## Steps

### 1. Initialize Go Module

- [x] Initialize Go module with `go mod init gocommender`

```bash
go mod init gocommender
```

### 2. Create Standard Go Project Structure

- [x] Create directory structure following llm-shared guidelines

Following llm-shared/examples/go-project.doc-validator.yml:

```plain
gocommender/
├── cmd/
│   └── server/          # HTTP server main entry point
├── internal/            # Private application code
│   ├── api/            # HTTP handlers and routing
│   ├── config/         # Configuration management
│   ├── models/         # Data structures
│   └── services/       # Business logic services
├── pkg/                # Public library code (if any)
├── build/              # Build artifacts
├── plan/               # This planning documentation
├── go.mod
├── go.sum
├── Taskfile.yml        # Task runner configuration
└── .env.example        # Environment variables template
```

### 3. Create Taskfile.yml

- [x] Create Taskfile.yml with required tasks per llm-shared guidelines

Must include required tasks per llm-shared guidelines:

- build
- build-linux
- build-ci
- test
- test-ci
- lint

Build tasks depend on test and lint. All artifacts go to `build/` directory.

### 4. Initialize Basic Files

- [ ] Create `.gitignore` for Go projects
- [ ] Create `.env.example` with required environment variables
- [x] Create basic `README.md`

### 5. Set Up Minimal Dependencies

- [x] Add essential dependencies to go.mod

Initial go.mod should include only essential dependencies:

- `github.com/spf13/viper` for configuration
- `modernc.org/sqlite` for database (CGO-free)

## Verification Steps

- [ ] **Structure Check**: All required directories exist

   ```bash
   ls -la cmd/ internal/ pkg/ build/ plan/
   ```

- [ ] **Go Module Validation**:

   ```bash
   go mod tidy
   go mod verify
   ```

- [ ] **Task Runner**:

   ```bash
   task build
   ```

   Should succeed and create binary in `build/` directory

- [ ] **Git Integration**:

   ```bash
   git status
   ```

   Should show proper `.gitignore` behavior

## Dependencies

None - this is the foundation step.

## Next Steps

Proceed to `02_data_models.md` to define the core data structures and database schema.

## Notes

- Follow llm-shared preference for standard library over third-party when possible
- Use `modernc.org/sqlite` instead of CGO-dependent SQLite drivers
- All build output goes to `build/` directory per guidelines
