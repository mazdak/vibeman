# Vibeman Frontend Specification

This document outlines the frontend implementation requirements for the Vibeman web UI, focusing on missing features and improvements needed to match the main specification.

## 1. Create New Worktree Functionality

### Requirements
- Implement a modal/dialog when "New Worktree" button is clicked
- Modal should contain:
  - **Worktree Name**: Text input for the worktree name
  - **Branch Selection**: Dropdown to select existing branch or option to create new
  - **New Branch Name**: Text input (shown only when "create new branch" is selected)
  - **Base Branch**: Dropdown to select which branch to base the new branch on (if creating new)
  - **Auto-start Container**: Checkbox to automatically start the container after creation (default: true)
  - **Repository**: Should be pre-selected based on current context

### API Integration
- Call the backend API to create the worktree
- Show loading state during creation
- Handle errors gracefully with user-friendly messages
- Refresh worktree list after successful creation

### UI Flow
1. User clicks "New Worktree" button
2. Modal opens with form
3. User fills in required fields
4. User clicks "Create"
5. Loading state shown
6. On success: Modal closes, worktree list refreshes, success notification shown
7. On error: Error message displayed in modal

## 2. Path Information Display

### Worktree Cards
- Add a new field showing the filesystem path of each worktree
- Display format: Use monospace font for paths
- Consider truncating long paths with ellipsis and full path on hover
- Example: `/Users/username/vibeman/worktrees/project-feature-123`

### Repository Cards  
- Add filesystem path where the repository is cloned
- Display similarly to worktree paths
- Example: `/Users/username/vibeman/repos/my-project`

## 3. Settings Page Enhancements

### Additional Configuration Fields

#### Server Configuration Section
- **Server Port**: Number input (default: 8080)
- **Web UI Port**: Number input (default: 8081)
- **Default Services Configuration**: File path input for services.toml location

#### Display Format
- Group these under a new "Server Configuration" card
- Include help text explaining each setting
- Show current values from backend config

## 4. Global Services Display

### Requirements
- Add a read-only section in Settings tab showing global services configuration
- Display information from the global services.toml file
- Show for each service:
  - Service name
  - Description
  - Compose file path
  - Service name in compose file
  - Current status (if running)

### UI Design
- Use a card-based layout similar to the shared services section
- Make it clear these are read-only global configurations
- Consider adding an info box explaining what global services are

## 5. AI Container Visibility (Future Enhancement)

### Concept
- Show AI container status for each worktree
- Display Claude Code connection status
- Show AI container logs separately from application logs
- Quick access to AI-specific terminal via xterm.js
- Ideally mobile friendly access to temrinal

### Implementation Notes
- This will be implemented in a future iteration
- Requires backend API support for AI container information
- Consider adding an "AI" icon/badge on worktree cards

## 6. Remove Worktree Functionality

### Requirements
- Enhance the existing delete button functionality on worktree cards
- Implement proper confirmation dialog with safety checks
- Show warnings for:
  - Uncommitted changes
  - Unstaged files
  - Unpushed commits
- Display these warnings in the confirmation dialog

### Confirmation Dialog
- Title: "Remove Worktree"
- Show worktree name and branch
- List any warnings (uncommitted/unstaged/unpushed changes)
- Two-step confirmation for worktrees with unsaved work
- Options:
  - "Cancel" - closes dialog
  - "Remove Anyway" - proceeds with deletion (shown in red)

### API Integration
- Call backend API to check worktree status before removal
- Handle the removal process with proper error handling
- Stop any running containers before removal
- Refresh worktree list after successful removal

## 7. Remove Repository Functionality

### Requirements
- Add a remove/delete option for repositories in the Repositories tab
- Implement confirmation dialog before removal
- Check if repository has active worktrees

### UI Implementation
- Add a delete button (trash icon) to each repository card
- Could also add a context menu or actions dropdown

### Confirmation Dialog
- Title: "Remove Repository"
- Show repository name and path
- **Important**: Check and display if there are active worktrees
- If worktrees exist:
  - Show warning: "This repository has X active worktrees"
  - List the worktree names
  - Require explicit confirmation
- Options:
  - "Cancel" - closes dialog
  - "Remove Repository Only" - removes from Vibeman tracking but keeps files

### Safety Checks
- Prevent removal if active worktrees exist (or require force flag)
- Show clear warnings about what will be deleted
- Differentiate between:
  - Removing from Vibeman tracking only
  - Actually deleting files from disk

### API Integration
- Backend should validate no active worktrees exist
- Handle both "untrack" and "delete" operations
- Return clear error messages if operation cannot be completed

## 8. General UI/UX Improvements

### Notifications
- Implement toast notifications for:
  - Successful operations (worktree created, service started, etc.)
  - Errors with actionable messages
  - Long-running operations progress

### Loading States
- Consistent loading indicators across all async operations
- Skeleton loaders for initial data fetching
- Optimistic updates where appropriate

### Error Handling
- User-friendly error messages
- Retry mechanisms for failed operations
- Clear indication of what went wrong and how to fix it

### Responsive Design
- Ensure all new features work well on tablet and mobile screens
- Consider collapsing paths on smaller screens
- Responsive modal sizes

## 7. Repository Standardization

### Terminology Update
- Rename all instances of "project" to "repository" in the UI
- Update API types if needed
- Ensure consistency across all components
- Update any user-facing messages

## Implementation Priority

1. **High Priority**
   - Create New Worktree functionality
   - Remove Worktree functionality (with safety checks)
   - Remove Repository functionality
   - Path information display
   - Settings page enhancements
   - Repository terminology standardization

2. **Medium Priority**
   - Global services display
   - Improved notifications
   - Better error handling

3. **Low Priority**
   - AI container visibility (future)
   - Additional UI polish

## Technical Considerations

### Data Fetching Architecture
**CRITICAL**: All frontend data fetching MUST use React Query (TanStack Query) with the OpenAPI-generated client.

#### Required Pattern for New Features
1. **Use Generated SDK**: All API calls must use the auto-generated SDK from `@/generated/api/sdk.gen`
2. **React Query Integration**: Use generated TanStack Query hooks from `@/generated/api/@tanstack/react-query.gen`
3. **Custom Hooks**: Wrap generated hooks in custom hooks for additional functionality
4. **Thin Wrapper**: Use `@/lib/api-client.ts` only for data transformation, never for business logic
5. **Legacy API**: Only use `@/lib/legacy-api.ts` for endpoints not yet in OpenAPI spec

#### Example Implementation
```typescript
// ✅ Correct approach for new features
export function useRepositories() {
  const query = useQuery(getRepositoriesOptions());
  
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await client.DELETE('/repositories/{id}', { path: { id } });
      if (response.error) throw new Error(response.error.error);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['getRepositories'] });
    },
  });

  return {
    repositories: query.data?.repositories || [],
    isLoading: query.isLoading,
    error: query.error,
    deleteRepository: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  };
}
```

#### Forbidden Patterns
- ❌ Manual fetch calls
- ❌ useState for server data
- ❌ Manual loading/error state management for server data
- ❌ Direct use of old API wrapper class

### General Guidelines
- Use existing UI components and patterns
- Maintain consistency with current design system
- Ensure all new features support both light and dark themes
- Follow existing code structure and conventions
- Add proper TypeScript types for all new data structures
- Include loading and error states for all async operations
