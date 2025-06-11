# Data Model Implementation Guide

This guide explains how to implement new data views and events in the Yesterday framework, using the user management system as a reference example.

## Architecture Overview

The Yesterday framework uses an event-driven architecture where:
- **Events** modify application state (write operations)
- **Data Views** query application state (read operations)
- Both are automatically synchronized across connected clients
- Data is cached efficiently with automatic invalidation

## JSON Field Naming Convention

**Important**: All JSON fields in Go structs should use camelCase naming to match TypeScript conventions. Database columns use snake_case, but JSON serialization should convert to camelCase using appropriate struct tags.

Example:
```go
type User struct {
    ID        int    `db:"id" json:"id"`
    Username  string `db:"username" json:"username"`
    CreatedAt string `db:"created_at" json:"createdAt"`  // Note: snake_case in DB, camelCase in JSON
}
```

## Implementing Data Views

Data views provide read-only access to application data with automatic caching and real-time updates.

### 1. Backend: Database Query Function

First, create a function to query your data:

```go
// In your state package (e.g., state/posts.go)
type Post struct {
    ID        int    `db:"id" json:"id"`
    Title     string `db:"title" json:"title"`
    Content   string `db:"content" json:"content"`
    CreatedAt string `db:"created_at" json:"createdAt"`
}

func GetPosts(db *sqlx.DB) ([]Post, error) {
    ret := []Post{}
    err := db.Select(&ret, "SELECT id, title, content, created_at FROM posts_v1 ORDER BY created_at DESC")
    if err != nil {
        return ret, fmt.Errorf("failed to select posts: %v", err)
    }
    return ret, nil
}
```

### 2. Backend: Register API Endpoint

Register the endpoint in your main.go file:

```go
// In main.go init() function
guest.RegisterHandler("/api/posts", func(params types.RequestParams) types.Response {
    ret, err := state.GetPosts(db)
    return guest.CreateResponse(map[string]any{
        "posts": ret,
    }, err, "Error fetching posts")
})
```

### 3. Frontend: TypeScript Types

Define TypeScript types that match your Go structs:

```tsx
// In dataviews/posts.tsx
export type Post = {
  id: number;
  title: string;
  content: string;
  createdAt: string;
};
```

### 4. Frontend: Custom Hook

Create a custom hook that wraps `useDataView`:

```tsx
// In dataviews/posts.tsx
import { useDataView } from "@tomyedwab/yesterday";

export function usePostsView(): [boolean, Post[]] {
  const [loading, response] = useDataView("api/posts");
  if (loading || response === null) {
    return [true, []];
  }
  return [false, response.posts];
}
```

### 5. Frontend: Using the Data View

Use your custom hook in React components:

```tsx
function PostsList() {
  const [loading, posts] = usePostsView();
  
  if (loading) {
    return <div>Loading posts...</div>;
  }
  
  return (
    <div>
      {posts.map((post) => (
        <div key={post.id}>
          <h3>{post.title}</h3>
          <p>{post.content}</p>
        </div>
      ))}
    </div>
  );
}
```

## Implementing Events

Events handle write operations and state changes in the application.

### 1. Backend: Define Event Type and Struct

```go
// In your state package (e.g., state/posts.go)
const PostCreatedEventType string = "CreatePost"

type PostCreatedEvent struct {
    events.GenericEvent
    Title   string `json:"title"`
    Content string `json:"content"`
}
```

### 2. Backend: Event Handler Function

Create a handler function that processes the event:

```go
func PostsHandleCreatedEvent(tx *sqlx.Tx, event *PostCreatedEvent) (bool, error) {
    guest.WriteLog(fmt.Sprintf("Creating post: %s", event.Title))
    
    _, err := tx.Exec(`
        INSERT INTO posts_v1 (title, content, created_at) 
        VALUES ($1, $2, datetime('now'))`,
        event.Title, event.Content)
    
    if err != nil {
        return false, fmt.Errorf("failed to insert post %s: %w", event.Title, err)
    }
    
    return true, nil
}
```

### 3. Backend: Register Event Handler

Register your event handler in main.go:

```go
// In main.go init() function
guest.RegisterEventHandler(state.PostCreatedEventType, state.PostsHandleCreatedEvent)
```

### 4. Backend: Database Initialization (if needed)

If you need to create tables, add an init event handler:

```go
func PostsHandleInitEvent(tx *sqlx.Tx, event *events.DBInitEvent) (bool, error) {
    _, err := tx.Exec(`
        CREATE TABLE posts_v1 (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            created_at TEXT NOT NULL
        )`)
    
    if err != nil {
        return false, fmt.Errorf("failed to create posts table: %w", err)
    }
    
    fmt.Println("Posts table initialized.")
    return true, nil
}
```

Don't forget to register the init handler:

```go
guest.RegisterEventHandler(events.DBInitEventType, state.PostsHandleInitEvent)
```

### 5. Frontend: Publishing Events

Use the connection dispatch to publish events:

```tsx
import { useConnectionDispatch, CreatePendingEvent } from "@tomyedwab/yesterday";

function CreatePostForm() {
  const connectDispatch = useConnectionDispatch();
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  
  const handleSubmit = () => {
    connectDispatch(
      CreatePendingEvent("createpost:", {
        type: "CreatePost",
        title: title,
        content: content,
      })
    );
    
    // Clear form
    setTitle("");
    setContent("");
  };
  
  return (
    <form onSubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
      <input 
        value={title} 
        onChange={(e) => setTitle(e.target.value)}
        placeholder="Title" 
      />
      <textarea 
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="Content"
      />
      <button type="submit">Create Post</button>
    </form>
  );
}
```

## Data View Parameters

Data views can accept parameters for filtering or customization:

### Backend with Parameters

```go
guest.RegisterHandler("/api/posts", func(params types.RequestParams) types.Response {
    // Extract query parameters
    categoryFilter := params.Query.Get("category")
    
    var ret []Post
    var err error
    
    if categoryFilter != "" {
        ret, err = state.GetPostsByCategory(db, categoryFilter)
    } else {
        ret, err = state.GetPosts(db)
    }
    
    return guest.CreateResponse(map[string]any{
        "posts": ret,
    }, err, "Error fetching posts")
})
```

Note: If your Post struct has fields that map from snake_case database columns, make sure to use camelCase in the JSON tags:

```go
type Post struct {
    ID         int    `db:"id" json:"id"`
    Title      string `db:"title" json:"title"`
    Content    string `db:"content" json:"content"`
    CreatedAt  string `db:"created_at" json:"createdAt"`
    CategoryID int    `db:"category_id" json:"categoryId"`
}
```

### Frontend with Parameters

```tsx
export function usePostsView(category?: string): [boolean, Post[]] {
  const params = category ? { category } : {};
  const [loading, response] = useDataView("api/posts", params);
  
  if (loading || response === null) {
    return [true, []];
  }
  return [false, response.posts];
}
```

## Best Practices

### Database Operations

1. **Use transactions**: Event handlers receive a transaction (`*sqlx.Tx`) - use it for all database operations
2. **Handle errors gracefully**: Return meaningful error messages
3. **Use prepared statements**: Protect against SQL injection
4. **Version your tables**: Use table names like `posts_v1` for schema evolution

### Event Design

1. **Make events granular**: One event per logical operation
2. **Include all necessary data**: Events should be self-contained
3. **Use descriptive event types**: Make event types easy to understand
4. **Log important events**: Use `guest.WriteLog()` for debugging

### Data Views

1. **Keep queries efficient**: Use appropriate indexes and LIMIT clauses
2. **Return consistent data structures**: Always return the same shape of data
3. **Handle empty results**: Gracefully handle cases where no data exists
4. **Use meaningful error messages**: Help with debugging

### Frontend Integration

1. **Create custom hooks**: Wrap `useDataView` for better type safety
2. **Handle loading states**: Always show appropriate loading indicators
3. **Use TypeScript types**: Maintain type safety between frontend and backend
4. **Batch related operations**: Group related UI updates together

## Testing Your Implementation

1. **Test the data view**: Verify data loads correctly in the frontend
2. **Test event publishing**: Ensure events trigger and update data
3. **Test error conditions**: Verify graceful handling of failures
4. **Test real-time updates**: Confirm data refreshes when events occur
5. **Test with multiple clients**: Verify synchronization across connections

## Common Patterns

### Master-Detail Views

```tsx
// Master view
const [loading, posts] = usePostsView();

// Detail view with parameter
const [detailLoading, post] = usePostView(selectedPostId);
```

### Conditional Events

```tsx
const handleDelete = (postId: number) => {
  if (confirm("Are you sure?")) {
    connectDispatch(
      CreatePendingEvent("deletepost:", {
        type: "DeletePost",
        postId: postId,
      })
    );
  }
};
```

### Optimistic Updates

The framework handles optimistic updates automatically - events are applied immediately in the UI and rolled back if they fail on the server.

This guide provides the foundation for implementing new data models in the Yesterday framework. The event-driven architecture ensures that all clients stay synchronized automatically while providing a clean separation between read and write operations.