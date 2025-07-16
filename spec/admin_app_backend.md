# Admin App Backend Specification

## Introduction

This specification defines the backend API and data management system for the Yesterday Admin application. The Admin app provides a web-based interface for managing users, applications, and access control rules within the Yesterday platform.

Reference: `design/admin_app_backend.md`

## System Overview

The Admin app backend is a Go-based HTTP server that provides RESTful APIs for managing system resources. It uses SQLite for data persistence and an event-driven architecture for state management.

## Core Components

### 1. Main Entry Point (`admin-main`)
**Reference:** `apps/admin/main.go:16-89`
**Implementation Status:** Implemented

The main entry point initializes the application, registers HTTP handlers, and sets up event listeners for database state management.

**Responsibilities:**
- Initialize applib framework with version "0.0.1"
- Register internal login endpoints (`/internal/dologin`, `/internal/checkAccess`)
- Register REST API endpoints for data access
- Register event handlers for database initialization and state management
- Initialize database and start HTTP server

### 2. Authentication System (`admin-auth`)
**Reference:** `apps/admin/handlers/login.go:17-49`
**Implementation Status:** Implemented

Handles admin user authentication using SHA256 password hashing with per-user salts.

**Endpoints:**
- `POST /internal/dologin` - Authenticate admin user
  - Request: `AdminLoginRequest{Username, Password}`
  - Response: `AdminLoginResponse{Success, UserID}`

**Security Features:**
- Per-user salt generation using UUID v4
- SHA256 password hashing (salt + password)
- Protection against timing attacks through consistent response timing

### 3. Access Control System (`admin-access`)
**Reference:** `apps/admin/handlers/checkaccess.go:15-33`
**Implementation Status:** Implemented

Provides endpoint for checking user access to applications based on configured rules.

**Endpoints:**
- `POST /internal/checkAccess` - Check user access to application
  - Request: `AccessRequest{UserID, ApplicationID}`
  - Response: `AccessResponse{AccessGranted}`

### 4. User Management (`admin-users`)
**Reference:** `apps/admin/state/users.go:13-214`
**Implementation Status:** Implemented

Manages user accounts with support for CRUD operations and password management.

**Data Model:**
```go
type User struct {
    ID           int    `db:"id" json:"id"`
    Username     string `db:"username" json:"username"`
    Salt         string `db:"salt" json:"-"`
    PasswordHash string `db:"password_hash" json:"-"`
}
```

**API Endpoints:**
- `GET /api/users` - List all users (returns ID and username only)

**Event Types:**
- `AddUser` - Create new user
- `UpdateUserPassword` - Change user password
- `DeleteUser` - Remove user (prevents deletion of admin user ID 1)
- `UpdateUser` - Update username (prevents changing admin username)

**Database Schema:**
```sql
CREATE TABLE users_v1 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    salt TEXT NOT NULL,
    password_hash TEXT NOT NULL
);
CREATE INDEX idx_users_username ON users_v1(username);
```

### 5. Application Management (`admin-applications`)
**Reference:** `apps/admin/state/applications.go:11-225`
**Implementation Status:** Implemented

Manages application instances with support for both regular and debug applications.

**Data Model:**
```go
type Application struct {
    InstanceID         string  `db:"instance_id" json:"instanceId"`
    AppID              string  `db:"app_id" json:"appId"`
    DisplayName        string  `db:"display_name" json:"displayName"`
    HostName           string  `db:"host_name" json:"hostName"`
    DebugPublishToken  *string `db:"debug_publish_token" json:"debugPublishToken,omitempty"`
    DebugStaticService *string `db:"debug_static_service" json:"debugStaticService,omitempty"`
}
```

**API Endpoints:**
- `GET /api/applications` - List all applications

**Event Types:**
- `CreateApplication` - Create regular application
- `CreateDebugApplication` - Create debug application with token
- `UpdateApplication` - Update application details
- `DeleteApplication` - Remove application (prevents deletion of core system apps)

**Database Schema:**
```sql
CREATE TABLE applications_v1 (
    instance_id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    host_name TEXT NOT NULL,
    debug_publish_token TEXT,
    debug_static_service TEXT
);
CREATE INDEX idx_applications_app_id ON applications_v1(app_id);
CREATE INDEX idx_applications_display_name ON applications_v1(display_name);
```

### 6. Access Rule Management (`admin-access-rules`)
**Reference:** `apps/admin/state/user_access_rules.go:10-244`
**Implementation Status:** Implemented

Manages fine-grained access control rules for users and applications.

**Data Model:**
```go
type UserAccessRule struct {
    ID            int         `db:"id" json:"id"`
    ApplicationID string      `db:"application_id" json:"applicationId"`
    RuleType      RuleType    `db:"rule_type" json:"ruleType"` // ACCEPT or DENY
    SubjectType   SubjectType `db:"subject_type" json:"subjectType"` // USER or GROUP
    SubjectID     string      `db:"subject_id" json:"subjectId"`
    CreatedAt     string      `db:"created_at" json:"createdAt"`
}
```

**API Endpoints:**
- `GET /api/user-access-rules` - List all access rules
- `GET /api/user-access-rules?applicationId={id}` - List rules for specific application

**Event Types:**
- `CreateUserAccessRule` - Create new access rule
- `DeleteUserAccessRule` - Remove access rule

**Access Rule Evaluation:**
Rules are evaluated in priority order:
1. USER ACCEPT/DENY rules (highest priority)
2. GROUP ACCEPT/DENY rules
3. Default: DENY access if no rule applies

**Database Schema:**
```sql
CREATE TABLE user_access_rules_v1 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    application_id TEXT NOT NULL,
    rule_type TEXT NOT NULL CHECK (rule_type IN ('ACCEPT', 'DENY')),
    subject_type TEXT NOT NULL CHECK (subject_type IN ('USER', 'GROUP')),
    subject_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (application_id) REFERENCES applications_v1(instance_id)
);
CREATE INDEX idx_user_access_rules_lookup
    ON user_access_rules_v1(application_id, subject_type, subject_id);
```

## Event System Integration

The backend uses an event-driven architecture for state management:

**Database Initialization Events:**
- `DBInitEventType` triggers table creation and seed data
- Creates admin user with password "admin"
- Creates default applications (login, admin)
- Creates default access rules for admin user

**State Management Events:**
- All CRUD operations are handled through events
- Event handlers ensure data consistency and foreign key relationships
- Cascade deletes are handled explicitly in event handlers

## Security Considerations

1. **Password Security:**
   - Per-user salt generation using UUID v4
   - SHA256 hashing with salt prefix
   - Password hashes never exposed in API responses

2. **Access Control:**
   - Fine-grained rule-based access control
   - Protection of core system applications
   - Protection of admin user from deletion/username change

3. **Input Validation:**
   - JSON request parsing with error handling
   - SQL injection prevention through parameterized queries
   - Foreign key constraints in database schema

## API Response Format

All API endpoints use consistent response format through `httputils.HandleAPIResponse`:
- Success: JSON data with HTTP 200
- Error: JSON error message with appropriate HTTP status code
- Internal errors return HTTP 500

## Testing Requirements

1. **Unit Tests:**
   - Test all event handlers with mock database transactions
   - Test access rule evaluation logic
   - Test password hashing and verification
   - Test admin user protection mechanisms

2. **Integration Tests:**
   - Test full CRUD cycles for users, applications, and access rules
   - Test cascade delete behavior
   - Test access control evaluation with complex rule sets

3. **Security Tests:**
   - Test SQL injection prevention
   - Test password security
   - Test unauthorized access attempts
