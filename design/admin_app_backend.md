# Admin App Backend - Technical Requirements Document

## Executive Summary

The Admin App Backend serves as the central management system for the Yesterday platform, providing secure administrative capabilities for user management, application configuration, and access control. This backend must support a web-based administrative interface while maintaining high security standards and operational reliability.

## Business Requirements

### 1. Administrative Access
- **BR-001**: Provide secure authentication for administrative users
- **BR-002**: Support role-based access to administrative functions
- **BR-003**: Prevent unauthorized access to sensitive operations
- **BR-004**: Maintain audit trail of administrative actions

### 2. User Lifecycle Management
- **BR-005**: Create and manage user accounts with secure password storage
- **BR-006**: Support password changes and account updates
- **BR-007**: Prevent deletion of critical system accounts
- **BR-008**: Provide user listing and search capabilities

### 3. Application Management
- **BR-009**: Register and configure new applications in the system
- **BR-010**: Support both production and debug application modes
- **BR-011**: Prevent modification of core system applications
- **BR-012**: Provide application discovery and listing

### 4. Access Control System
- **BR-013**: Implement fine-grained access control between users and applications
- **BR-014**: Support both user-specific and group-based permissions
- **BR-015**: Provide priority-based rule evaluation
- **BR-016**: Allow real-time access checking

## Technical Architecture Requirements

### 1. System Architecture
- **TA-001**: Event-driven architecture for state consistency
- **TA-002**: RESTful API design with JSON payloads
- **TA-003**: SQLite database for persistence
- **TA-004**: Go-based implementation for performance and reliability

### 2. Security Architecture
- **TA-005**: Per-user salt generation for password hashing
- **TA-006**: SHA256-based password storage with salt prefix
- **TA-007**: Protection against SQL injection through parameterized queries
- **TA-008**: Input validation and sanitization
- **TA-009**: Protection of system-critical resources

### 3. Data Architecture
- **TA-010**: Normalized database schema with foreign key constraints
- **TA-011**: Indexed columns for query performance
- **TA-012**: Transactional integrity for state changes
- **TA-013**: Cascade delete handling for referential integrity

## Functional Requirements

### 1. Authentication System
- **FR-001**: Authenticate users with username/password credentials
- **FR-002**: Return user ID upon successful authentication
- **FR-003**: Provide clear success/failure responses
- **FR-004**: Prevent timing attacks through consistent response timing

### 2. User Management
- **FR-005**: Create new user accounts with unique usernames
- **FR-006**: Update existing user account information
- **FR-007**: Change user passwords with new salt generation
- **FR-008**: Delete user accounts with cascade cleanup
- **FR-009**: List all users with basic information
- **FR-010**: Retrieve individual user details

### 3. Application Management
- **FR-011**: Create new application instances
- **FR-012**: Create debug applications with special tokens
- **FR-013**: Update application configuration
- **FR-014**: Delete applications with cascade cleanup
- **FR-015**: List all applications with full details
- **FR-016**: Retrieve individual application details

### 4. Access Control Management
- **FR-017**: Create access rules linking users/groups to applications
- **FR-018**: Define rule types (ACCEPT/DENY) and subject types (USER/GROUP)
- **FR-019**: Delete access rules
- **FR-020**: List all access rules with filtering capabilities
- **FR-021**: Evaluate access rules in priority order
- **FR-022**: Provide real-time access checking

### 5. System Initialization
- **FR-023**: Initialize database schema on first run
- **FR-024**: Create default admin user with known credentials
- **FR-025**: Create default system applications
- **FR-026**: Create default access rules for admin user

## Non-Functional Requirements

### 1. Performance
- **NFR-001**: API response times under 100ms for standard queries
- **NFR-002**: Support for at least 1000 concurrent users
- **NFR-003**: Efficient database queries with proper indexing
- **NFR-004**: Minimal memory footprint for long-running operations

### 2. Reliability
- **NFR-005**: 99.9% uptime for administrative functions
- **NFR-006**: Graceful handling of database connection failures
- **NFR-007**: Transaction rollback on error conditions
- **NFR-008**: Consistent state across system restarts

### 3. Security
- **NFR-009**: Protection against common web vulnerabilities (OWASP Top 10)
- **NFR-010**: Secure password storage that resists rainbow table attacks
- **NFR-011**: Prevention of privilege escalation attacks
- **NFR-012**: Protection of system-critical resources from modification

### 4. Maintainability
- **NFR-013**: Clear separation of concerns between handlers and state
- **NFR-014**: Comprehensive logging for debugging and audit
- **NFR-015**: Consistent error handling and reporting
- **NFR-016**: Well-documented API endpoints and data models

## API Design Requirements

### 1. RESTful Endpoints
- **API-001**: `POST /internal/dologin` - User authentication
- **API-002**: `POST /internal/checkAccess` - Access verification
- **API-003**: `GET /api/users` - List all users
- **API-004**: `GET /api/applications` - List all applications
- **API-005**: `GET /api/user-access-rules` - List access rules (with optional filtering)

### 2. Request/Response Formats
- **API-006**: JSON-based request and response payloads
- **API-007**: Consistent error response format
- **API-008**: Appropriate HTTP status codes
- **API-009**: Clear and descriptive error messages

### 3. Data Validation
- **API-010**: Input validation for all request parameters
- **API-011**: Type checking and sanitization
- **API-012**: Business rule validation (e.g., unique usernames)

## Database Design Requirements

### 1. Schema Design
- **DB-001**: Normalized schema with proper relationships
- **DB-002**: Foreign key constraints for referential integrity
- **DB-003**: Indexed columns for query performance
- **DB-004**: Appropriate data types for each field

### 2. Tables
- **DB-005**: `users_v1` - User account storage
- **DB-006**: `applications_v1` - Application configuration
- **DB-007**: `user_access_rules_v1` - Access control rules

### 3. Security
- **DB-008**: Password hashes never stored in plain text
- **DB-009**: Salts generated using cryptographically secure random sources
- **DB-010**: Protection against SQL injection through parameterized queries

## Event System Requirements

### 1. Event Types
- **EVT-001**: Database initialization events
- **EVT-002**: User lifecycle events (create, update, delete)
- **EVT-003**: Application lifecycle events (create, update, delete)
- **EVT-004**: Access rule lifecycle events (create, delete)

### 2. Event Handling
- **EVT-005**: Transactional event processing
- **EVT-006**: Rollback on error conditions
- **EVT-007**: Idempotent event handling
- **EVT-008**: Comprehensive logging of event processing

## Testing Requirements

### 1. Unit Testing
- **TST-001**: Test all event handlers with mock transactions
- **TST-002**: Test password hashing and verification
- **TST-003**: Test access rule evaluation logic
- **TST-004**: Test system protection mechanisms

### 2. Integration Testing
- **TST-005**: Test complete user lifecycle
- **TST-006**: Test application lifecycle with access rules
- **TST-007**: Test cascade delete behavior
- **TST-008**: Test system initialization

### 3. Security Testing
- **TST-009**: Test SQL injection prevention
- **TST-010**: Test password security
- **TST-011**: Test unauthorized access attempts
- **TST-012**: Test system resource protection

## Deployment Requirements

### 1. Configuration
- **DEP-001**: Environment-based configuration
- **DEP-002**: Database file location configuration
- **DEP-003**: Port and binding configuration
- **DEP-004**: Logging level configuration

### 2. Monitoring
- **DEP-005**: Health check endpoint
- **DEP-006**: Performance metrics collection
- **DEP-007**: Error rate monitoring
- **DEP-008**: Security event logging

## Future Considerations

### 1. Scalability
- **FUT-001**: Support for multiple database backends
- **FUT-002**: Horizontal scaling capabilities
- **FUT-003**: Caching layer for frequently accessed data
- **FUT-004**: API rate limiting

### 2. Enhanced Security
- **FUT-005**: Multi-factor authentication support
- **FUT-006**: Session management with expiration
- **FUT-007**: Audit logging for all administrative actions
- **FUT-008**: API key authentication for service-to-service communication

### 3. Advanced Features
- **FUT-009**: Bulk operations for user management
- **FUT-010**: Application templates for common configurations
- **FUT-011**: Advanced access rule conditions (time-based, IP-based)
- **FUT-012**: User groups and hierarchical permissions