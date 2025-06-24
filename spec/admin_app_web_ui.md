# Admin Web UI Technical Specification

**Design Document**: [design/admin_app_web_ui.md](../design/admin_app_web_ui.md)

## Introduction

This specification defines the implementation of the admin web UI for the Yesterday framework, providing a comprehensive management interface for users, applications, and access rules. The UI is built as a modern React application using TypeScript, Vite, and Chakra UI v3.

## References

Implementation references:
- Main NexusHub design: `design/nexushub.md`
- Process management: `design/processes.md` 
- Data model documentation: `doc/data_model.md`
- Chakra UI components: `doc/chakra-components.txt`

Note: No specific design document exists for the admin UI - this specification documents the implemented solution.

## Implementation Tasks

### Task `web-project-setup`: Project Setup and Dependencies
**Reference**: Implementation in `apps/admin/web`
**Implementation status**: Completed (2025-06-24)
**Files**: 
- `apps/admin/web/package.json`
- `apps/admin/web/vite.config.ts`
- `apps/admin/web/tsconfig.json`

**Details**:
React TypeScript project using Vite as build tool with the following key dependencies:
- React 19.1.0 with TypeScript support
- Chakra UI 3.20.0 for component library
- React Icons for iconography
- Emotion for CSS-in-JS styling
- Next Themes for theme support

Build configuration includes TypeScript compilation, ESLint for code quality, and Vite for development server and production builds.

### Task `web-app-structure`: Application Structure and Entry Point
**Reference**: Main application files
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/App.tsx`
- `apps/admin/web/src/main.tsx`
- `apps/admin/web/index.html`

**Details**:
Main application structure with proper provider hierarchy:
- Chakra UI Provider for theming and components
- ConnectionStateProvider from Yesterday framework for backend connectivity
- MainLayout component as primary interface
- Toaster component for notifications

Entry point configured for React 19 with proper TypeScript integration.

### Task `web-ui-components`: UI Component System
**Reference**: Chakra UI v3 component system
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/components/ui/provider.tsx`
- `apps/admin/web/src/components/ui/toaster.tsx`
- `apps/admin/web/src/components/ui/*.tsx`

**Details**:
Complete UI component system built on Chakra UI v3:
- Theme provider with system configuration
- Toast notification system for user feedback
- Reusable UI components following Chakra v3 patterns
- Proper TypeScript interfaces for all components

### Task `web-layout-system`: Main Layout and Navigation
**Reference**: Tab-based navigation interface
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/components/layout/MainLayout.tsx`
- `apps/admin/web/src/components/layout/ConnectionStateHeader.tsx`

**Details**:
Primary application layout with:
- Connection state header showing backend connectivity status
- Tab-based navigation using Chakra UI Tabs component
- Two main tabs: "Users" and "Applications"
- Responsive design with proper spacing and padding
- Icon integration using React Icons (LuUsers, LuCog)

### Task `web-data-layer`: Data Views and API Integration
**Reference**: Yesterday framework data layer
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/dataviews/users.tsx`
- `apps/admin/web/src/dataviews/applications.tsx`
- `apps/admin/web/src/dataviews/userAccessRules.tsx`

**Details**:
Complete data layer integration with Yesterday framework:
- `useUsersView()` hook for user data fetching
- `useApplicationsView()` hook for application data fetching
- `useUserAccessRulesView()` and `useAllUserAccessRulesView()` hooks for access rules
- TypeScript interfaces for all data types (User, Application, UserAccessRule)
- Proper loading state management and error handling

### Task `web-user-actions`: User Management Actions
**Reference**: User CRUD operations
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/dataviews/userActions.tsx`

**Details**:
Complete user management action hooks:
- `useCreateUser()` - Create new users with password validation
- `useUpdateUser()` - Update user details with admin protection
- `useDeleteUser()` - Delete users with admin safeguards
- `useUpdateUserPassword()` - Secure password updates
- TypeScript interfaces for all action parameters
- Error handling and success notifications

### Task `web-application-actions`: Application Management Actions
**Reference**: Application CRUD operations
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/dataviews/applicationActions.tsx`

**Details**:
Complete application management action hooks:
- `useCreateApplication()` - Create new applications with validation
- `useUpdateApplication()` - Update application details
- `useDeleteApplication()` - Delete applications with protection for system apps
- TypeScript interfaces for all action parameters
- UUID generation for instance IDs
- Field validation and error handling

### Task `web-access-rule-actions`: Access Rule Management Actions
**Reference**: User access rule CRUD operations
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/dataviews/userAccessRuleActions.tsx`

**Details**:
Complete access rule management action hooks:
- `useCreateUserAccessRule()` - Create access rules with validation
- `useDeleteUserAccessRule()` - Remove access rules
- TypeScript interfaces for RuleType ("ACCEPT" | "DENY") and SubjectType ("USER" | "GROUP")
- Application-specific rule management
- Error handling and validation

### Task `web-users-tab`: User Management Interface
**Reference**: User management UI components
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/components/users/UsersTab.tsx`
- `apps/admin/web/src/components/users/UsersList.tsx`
- `apps/admin/web/src/components/users/CreateUserModal.tsx`
- `apps/admin/web/src/components/users/EditUserModal.tsx`
- `apps/admin/web/src/components/users/DeleteUserModal.tsx`
- `apps/admin/web/src/components/users/ChangePasswordModal.tsx`

**Details**:
Complete user management interface with:
- Professional table view using Chakra UI Table components
- Create user modal with username and password fields
- Edit user modal with admin user protection
- Delete user modal with confirmation and admin safeguards
- Change password modal with strength validation (8+ chars, uppercase, lowercase, number)
- Password confirmation validation and show/hide toggles
- Loading states and error handling throughout
- Toast notifications for user feedback

### Task `web-applications-tab`: Application Management Interface
**Reference**: Application management UI components
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/components/applications/ApplicationsTab.tsx`
- `apps/admin/web/src/components/applications/ApplicationsList.tsx`
- `apps/admin/web/src/components/applications/CreateApplicationModal.tsx`
- `apps/admin/web/src/components/applications/EditApplicationModal.tsx`
- `apps/admin/web/src/components/applications/DeleteApplicationModal.tsx`

**Details**:
Complete application management interface with:
- Professional table view with application details
- Create application modal with App ID, display name, hostname, and database name
- Edit application modal with system application protection
- Delete application modal with confirmation and access rule warnings
- Field validation (App ID format, required fields)
- UUID auto-generation for instance IDs
- Loading states and success notifications
- Integration with access rules management

### Task `web-access-rules-management`: Access Rules Management Interface
**Reference**: Access control UI components
**Implementation status**: Completed (2025-06-24)
**Files**:
- `apps/admin/web/src/components/access-rules/UserAccessRulesList.tsx`
- `apps/admin/web/src/components/access-rules/CreateUserAccessRuleModal.tsx`
- `apps/admin/web/src/components/access-rules/EditUserAccessRuleModal.tsx`
- `apps/admin/web/src/components/access-rules/DeleteUserAccessRuleModal.tsx`
- `apps/admin/web/src/components/access-rules/index.ts`

**Details**:
Complete access rules management with:
- Access rules listing with application-specific filtering
- Create access rule modal with rule type and subject type selection
- Edit access rule functionality with validation
- Delete access rule confirmation
- Integration with applications management interface
- TypeScript interfaces for rule types (ACCEPT/DENY) and subject types (USER/GROUP)
- Proper error handling and user feedback

### Task `web-integration-testing`: Framework Integration
**Reference**: Yesterday framework integration
**Implementation status**: Completed (2025-06-24)
**Files**: All components integrate with Yesterday framework

**Details**:
Complete integration with Yesterday framework:
- ConnectionStateProvider for backend connectivity monitoring
- useDataView hooks for data fetching from framework APIs
- Event-driven architecture for all CRUD operations
- Real-time updates through framework event system
- Proper error handling and loading states
- Toast notifications integrated with UI feedback system

## Implementation Summary

**Total Implementation Status**: Fully Completed (2025-06-24)

The admin web UI is a comprehensive React TypeScript application providing complete management capabilities for the Yesterday framework:

### Architecture
- **Frontend**: React 19.1.0 with TypeScript
- **Build Tool**: Vite 6.3.5 with hot reloading
- **UI Library**: Chakra UI 3.20.0 with modern component patterns
- **State Management**: React hooks with Yesterday framework integration
- **Icons**: React Icons with Lucide icon set

### Features Implemented
1. **User Management**: Complete CRUD operations with password management and admin protection
2. **Application Management**: Full application lifecycle management with system protection
3. **Access Rules Management**: Comprehensive access control with rule-based permissions
4. **Professional UI**: Modern, responsive interface with loading states and notifications
5. **Type Safety**: Full TypeScript implementation with proper interfaces
6. **Framework Integration**: Seamless integration with Yesterday event system and data layer

### File Structure
```
apps/admin/web/
├── src/
│   ├── components/
│   │   ├── access-rules/     # Access rules management (5 files)
│   │   ├── applications/     # Application management (5 files)
│   │   ├── layout/          # Main layout and navigation (2 files)
│   │   ├── ui/              # Base UI components (4 files)
│   │   └── users/           # User management (6 files)
│   ├── dataviews/           # Data layer and actions (6 files)
│   ├── App.tsx              # Main application component
│   └── main.tsx             # Application entry point
├── package.json             # Dependencies and scripts
├── vite.config.ts           # Build configuration
└── tsconfig.json            # TypeScript configuration
```

### Key Technical Achievements
- **Modern React Patterns**: Functional components with hooks
- **Type Safety**: Comprehensive TypeScript interfaces
- **Component Architecture**: Modular, reusable components
- **State Management**: Efficient state handling with React hooks
- **User Experience**: Professional UI with proper loading states and feedback
- **Security**: Admin user protection and system application safeguards
- **Performance**: Optimized builds with Vite and proper code splitting

The implementation provides a production-ready admin interface that fully integrates with the Yesterday framework's architecture and provides comprehensive management capabilities for users, applications, and access control.
