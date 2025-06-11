# Admin UI Implementation Plan

This document outlines the complete implementation plan for the Yesterday Admin UI, including user management and application management functionality.

## References

Data model implementation details are documented in `doc/data_model.md`

Current Chakra UI documentation is found at `doc/chakra-ui.txt`

To verify the backend changes, run `make serve` in the workspace directory.

## Overview

The admin UI will provide a comprehensive interface for managing users and applications in the Yesterday framework. It will feature a tabbed interface with dedicated sections for user management and application management.

## Architecture

### Frontend Structure
```
src/
├── components/
│   ├── ui/           # Chakra UI provider and base components
│   ├── layout/       # Main layout components (tabs, navigation)
│   ├── users/        # User management components
│   └── applications/ # Application management components
├── dataviews/        # Data view hooks
├── types/            # TypeScript type definitions
└── utils/            # Utility functions
```

### Backend Structure
- Event handlers for CRUD operations
- Data view endpoints for fetching data
- Validation and error handling
- Database schema updates

## Implementation Plan

### ✅ Phase 1: Backend Data Layer (COMPLETED)

#### ✅ 1.1 User Management Events (IMPLEMENTED)

**✅ Events implemented:**
- ✅ `UpdateUserPassword` - Change user password
- ✅ `DeleteUser` - Remove user from system (with admin protection)
- ✅ `UpdateUser` - Modify user details (with admin username protection)

**Event Structures:**
```go
type UpdateUserPasswordEvent struct {
    events.GenericEvent
    UserID      int    `json:"userId"`
    NewPassword string `json:"newPassword"`
}

type DeleteUserEvent struct {
    events.GenericEvent
    UserID int `json:"userId"`
}

type UpdateUserEvent struct {
    events.GenericEvent
    UserID   int    `json:"userId"`
    Username string `json:"username"`
}
```

#### ✅ 1.2 Application Management Events (IMPLEMENTED)

**✅ Events implemented:**
- ✅ `CreateApplication` - Add new application (with UUID generation)
- ✅ `UpdateApplication` - Modify application details
- ✅ `DeleteApplication` - Remove application (with core system protection)

**Event Structures:**
```go
type CreateApplicationEvent struct {
    events.GenericEvent
    AppID       string `json:"appId"`
    DisplayName string `json:"displayName"`
    HostName    string `json:"hostName"`
    DBName      string `json:"dbName"`
}

type UpdateApplicationEvent struct {
    events.GenericEvent
    InstanceID  string `json:"instanceId"`
    AppID       string `json:"appId"`
    DisplayName string `json:"displayName"`
    HostName    string `json:"hostName"`
    DBName      string `json:"dbName"`
}

type DeleteApplicationEvent struct {
    events.GenericEvent
    InstanceID string `json:"instanceId"`
}
```

#### ✅ 1.3 User Access Rules Events (IMPLEMENTED)

**✅ Events implemented:**
- ✅ `CreateUserAccessRule` - Add access rule
- ✅ `DeleteUserAccessRule` - Remove access rule

**Event Structures:**
```go
type CreateUserAccessRuleEvent struct {
    events.GenericEvent
    ApplicationID string      `json:"applicationId"`
    RuleType      RuleType    `json:"ruleType"`
    SubjectType   SubjectType `json:"subjectType"`
    SubjectID     string      `json:"subjectId"`
}

type DeleteUserAccessRuleEvent struct {
    events.GenericEvent
    RuleID int `json:"ruleId"`
}
```

#### ✅ 1.4 Data View Endpoints (IMPLEMENTED)

**✅ Endpoints implemented:**
- ✅ `/api/applications` - List all applications (sorted by display name)
- ✅ `/api/user-access-rules` - List access rules with optional applicationId query parameter filter

### Phase 2: Frontend Components

#### 2.1 Main Layout Component

**File: `src/components/layout/MainLayout.tsx`**
- Chakra UI Tabs component for navigation
- Responsive design
- Header with connection status
- Tab panels for Users and Applications

#### 2.2 User Management Components

**File: `src/components/users/UsersTab.tsx`**
- Main container for user management
- User list with search and filtering
- Add user form
- User actions (edit, delete, change password)

**File: `src/components/users/UsersList.tsx`**
- Table/list view of users
- Action buttons for each user
- Loading and empty states

**File: `src/components/users/AddUserForm.tsx`**
- Form for creating new users
- Username validation
- Integration with CreatePendingEvent

**File: `src/components/users/EditUserModal.tsx`**
- Modal for editing user details
- Username modification
- Form validation

**File: `src/components/users/ChangePasswordModal.tsx`**
- Modal for changing user passwords
- Password strength validation
- Confirmation dialog

**File: `src/components/users/DeleteUserDialog.tsx`**
- Confirmation dialog for user deletion
- Warning about irreversible action

#### 2.3 Application Management Components

**File: `src/components/applications/ApplicationsTab.tsx`**
- Main container for application management
- Application list with details
- Add application form
- Application actions

**File: `src/components/applications/ApplicationsList.tsx`**
- Grid/card view of applications
- Application status indicators
- Action buttons

**File: `src/components/applications/AddApplicationForm.tsx`**
- Form for creating new applications
- Field validation
- UUID generation for instance ID

**File: `src/components/applications/EditApplicationModal.tsx`**
- Modal for editing application details
- All application fields editable
- Validation for hostnames and paths

**File: `src/components/applications/DeleteApplicationDialog.tsx`**
- Confirmation dialog for application deletion
- Warning about access rules cleanup

**File: `src/components/applications/AccessRulesSection.tsx`**
- Section showing access rules for each application
- Add/remove access rules functionality
- User and group rule management

#### 2.4 Data View Hooks

**File: `src/dataviews/applications.tsx`**
```tsx
export type Application = {
  instanceId: string;
  appId: string;
  displayName: string;
  hostName: string;
  dbName: string;
};

export function useApplicationsView(): [boolean, Application[]] {
  const [loading, response] = useDataView("api/applications");
  if (loading || response === null) {
    return [true, []];
  }
  return [false, response.applications];
}
```

**File: `src/dataviews/userAccessRules.tsx`**
```tsx
export type UserAccessRule = {
  id: number;
  applicationId: string;
  ruleType: "ACCEPT" | "DENY";
  subjectType: "USER" | "GROUP";
  subjectId: string;
  createdAt: string;
};

export function useUserAccessRulesView(applicationId?: string): [boolean, UserAccessRule[]] {
  const params = applicationId ? { applicationId: applicationId } : {};
  const [loading, response] = useDataView("api/user-access-rules", params);
  if (loading || response === null) {
    return [true, []];
  }
  return [false, response.rules];
}
```

### Phase 3: Enhanced Features

#### 3.1 Search and Filtering
- User search by username
- Application search by name/hostname
- Filter access rules by type/subject

#### 3.2 Validation and Error Handling
- Form validation with Chakra UI
- Error toasts for failed operations
- Loading states for all operations

#### 3.3 Confirmation Dialogs
- Delete confirmations
- Bulk operation confirmations
- Destructive action warnings

#### 3.4 Responsive Design
- Mobile-friendly layout
- Collapsible sidebars
- Responsive tables and forms

## Implementation Steps

### ✅ Step 1: Backend Event Handlers (COMPLETED)

1. **✅ Updated `state/users.go`:**
   - ✅ Added password hashing utilities
   - ✅ Implemented `UsersHandleUpdatePasswordEvent`
   - ✅ Implemented `UsersHandleDeleteEvent`
   - ✅ Implemented `UsersHandleUpdateEvent`
   - ✅ Added validation for username uniqueness and admin user protection

2. **✅ Updated `state/applications.go` event handlers:**
   - ✅ Implemented `ApplicationsHandleCreateEvent`
   - ✅ Implemented `ApplicationsHandleUpdateEvent`
   - ✅ Implemented `ApplicationsHandleDeleteEvent`
   - ✅ Added UUID generation utilities
   - ✅ Added protection for core system applications

3. **✅ Updated `state/user_access_rules.go`:**
   - ✅ Implemented `UserAccessRulesHandleCreateEvent`
   - ✅ Implemented `UserAccessRulesHandleDeleteEvent`
   - ✅ Added rule validation logic

4. **✅ Updated `main.go`:**
   - ✅ Registered all new event handlers
   - ✅ Registered new data view endpoints
   - ✅ Added proper error handling

### ✅ Step 2: Data View Endpoints (COMPLETED)

1. **✅ Applications endpoint:**
   - ✅ Implemented `GetApplications()` function
   - ✅ Registered `/api/applications` endpoint
   - ✅ Added sorting by display name

2. **✅ User Access Rules endpoint:**
   - ✅ Enhanced existing functions for filtering
   - ✅ Registered `/api/user-access-rules` endpoint
   - ✅ Added support for application ID filtering via query parameters

### Step 3: Frontend Data Layer (1 day)

1. **Create data view hooks:**
   - `useApplicationsView()`
   - `useUserAccessRulesView()`
   - Update existing `useUsersView()` if needed

2. **Define TypeScript types:**
   - Export all necessary types
   - Ensure type safety across components

### Step 4: Core UI Components (3-4 days)

1. **Main Layout:**
   - Implement tabbed interface
   - Add header with connection status
   - Style with Chakra UI theme

2. **Users Tab:**
   - User list with actions
   - Add user form
   - Edit/delete functionality
   - Password change modal

3. **Applications Tab:**
   - Application cards/list
   - Add application form
   - Edit/delete functionality
   - Access rules management

### Step 5: Advanced Features (2-3 days)

1. **Search and Filtering:**
   - Implement search functionality
   - Add filter controls
   - Optimize performance

2. **Validation and UX:**
   - Add form validation
   - Implement error handling
   - Add loading states
   - Create confirmation dialogs

3. **Polish and Testing:**
   - Responsive design testing
   - Cross-browser compatibility
   - User experience refinements

## ✅ Database Schema Updates (COMPLETED)

### ✅ Additional Indexes (IMPLEMENTED)
```sql
-- Improve performance for user lookups
CREATE INDEX idx_users_username ON users_v1(username);

-- Improve performance for application lookups
CREATE INDEX idx_applications_app_id ON applications_v1(app_id);
CREATE INDEX idx_applications_display_name ON applications_v1(display_name);
```

### Constraints and Validation
- Ensure username uniqueness
- Validate application hostnames
- Prevent deletion of admin user
- Cascade delete access rules when applications are deleted

## Security Considerations

1. **Authentication:**
   - Verify admin permissions for all operations
   - Session validation for sensitive operations

2. **Input Validation:**
   - Sanitize all user inputs
   - Validate hostnames and paths
   - Prevent SQL injection

3. **Password Security:**
   - Strong password requirements
   - Secure password hashing (already implemented)
   - Password change confirmation

4. **Access Control:**
   - Prevent users from deleting themselves
   - Require confirmation for destructive operations
   - Audit trail for administrative actions

## Testing Strategy

### Backend Testing
- Unit tests for event handlers
- Integration tests for data views
- Error handling validation
- Database constraint testing

### Frontend Testing
- Component unit tests
- Integration tests for data flows
- User interaction testing
- Responsive design testing

### End-to-End Testing
- Complete user workflows
- Error scenario handling
- Multi-user scenarios
- Performance testing

## Performance Considerations

1. **Database Optimization:**
   - Proper indexing strategy
   - Efficient query patterns
   - Connection pooling

2. **Frontend Optimization:**
   - Lazy loading for large lists
   - Debounced search inputs
   - Efficient re-rendering
   - Proper caching strategies

3. **Real-time Updates:**
   - Optimize WebSocket usage
   - Minimize unnecessary updates
   - Efficient diff calculations

## Deployment and Rollout

1. **Staging Environment:**
   - Deploy to staging for testing
   - User acceptance testing
   - Performance validation

2. **Production Deployment:**
   - Database migration scripts
   - Feature flag implementation
   - Rollback procedures

3. **Monitoring:**
   - Error tracking
   - Performance monitoring
   - User activity logging

## Future Enhancements

1. **Bulk Operations:**
   - Bulk user creation/deletion
   - Bulk access rule management
   - CSV import/export

2. **Advanced Access Control:**
   - Role-based permissions
   - Conditional access rules
   - Time-based access controls

3. **Audit and Logging:**
   - Comprehensive audit trail
   - Administrative action logging
   - Compliance reporting

4. **Advanced UI Features:**
   - Dark mode support
   - Customizable dashboards
   - Advanced filtering options
   - Data visualization

## ✅ Backend Implementation Summary (COMPLETED)

The backend portion of the admin UI has been successfully implemented with the following features:

### User Management Backend
- **Password Updates**: Users can have their passwords changed with secure salt generation and SHA-256 hashing
- **User Deletion**: Users can be deleted with cascade deletion of their access rules (admin user protected)
- **User Updates**: Usernames can be modified with admin username protection
- **Enhanced Security**: Admin user (ID=1) cannot be deleted or have username changed from "admin"

### Application Management Backend
- **Application Creation**: New applications can be created with auto-generated UUID instance IDs
- **Application Updates**: All application fields (appId, displayName, hostName, dbName) can be modified
- **Application Deletion**: Applications can be deleted with cascade deletion of access rules
- **System Protection**: Core system applications (login and admin services) cannot be deleted

### User Access Rules Management Backend
- **Rule Creation**: New access rules can be created for users or groups with ACCEPT/DENY permissions
- **Rule Deletion**: Access rules can be removed by rule ID
- **Enhanced Querying**: Rules can be fetched for all applications or filtered by specific application

### Data View Endpoints
- **`/api/users`**: Lists all users (existing, enhanced with proper JSON tags)
- **`/api/applications`**: Lists all applications sorted by display name
- **`/api/user-access-rules`**: Lists access rules with optional `applicationId` query parameter filtering

### Database Enhancements
- **Performance Indexes**: Added indexes on `users_v1.username`, `applications_v1.app_id`, and `applications_v1.display_name`
- **JSON Compatibility**: All struct fields use proper camelCase JSON tags for frontend compatibility
- **Cascade Operations**: Proper cascade deletion of related records (access rules when users/applications are deleted)

### Event System Integration
All new functionality is properly integrated with the Yesterday event system:
- **12 new event handlers** registered and implemented
- **3 new data view endpoints** with proper error handling
- **Type-safe event structures** with validation and error handling
- **Database transaction support** for all operations

The backend is now ready for frontend integration and provides a complete API for user management, application management, and access control rule management.

This implementation plan provides a comprehensive roadmap for building a full-featured admin UI that integrates seamlessly with the Yesterday framework's event-driven architecture while providing an excellent user experience through Chakra UI components.
