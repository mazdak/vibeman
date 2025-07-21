---
description: Use Bun instead of Node.js, npm, pnpm, or vite.
globs: "*.ts, *.tsx, *.html, *.css, *.js, *.jsx, package.json"
alwaysApply: false
---

Default to using Bun instead of Node.js.

- Use `bun <file>` instead of `node <file>` or `ts-node <file>`
- Use `bun test` instead of `jest` or `vitest`
- Use `bun build <file.html|file.ts|file.css>` instead of `webpack` or `esbuild`
- Use `bun install` instead of `npm install` or `yarn install` or `pnpm install`
- Use `bun run <script>` instead of `npm run <script>` or `yarn run <script>` or `pnpm run <script>`
- Bun automatically loads .env, so don't use dotenv.

## APIs

- `Bun.serve()` supports WebSockets, HTTPS, and routes. Don't use `express`.
- `bun:sqlite` for SQLite. Don't use `better-sqlite3`.
- `Bun.redis` for Redis. Don't use `ioredis`.
- `Bun.sql` for Postgres. Don't use `pg` or `postgres.js`.
- `WebSocket` is built-in. Don't use `ws`.
- Prefer `Bun.file` over `node:fs`'s readFile/writeFile
- Bun.$`ls` instead of execa.

## Frontend
- We use `shadcn/ui`
- We use Tailwind CSS 4
- We support dark mode and light mode
- Follow the design patterns of existing pages and components

### Data Fetching Architecture
**CRITICAL**: This frontend MUST use React Query (TanStack Query) strictly with a thin wrapper around the OpenAPI-generated client.

#### React Query Principles
1. **Generated SDK Integration**: Use the auto-generated SDK from OpenAPI spec:
   ```typescript
   // ✅ Use generated SDK functions
   import { getWorktrees, postWorktrees } from '@/generated/api/sdk.gen';
   
   // ✅ Use generated TanStack Query hooks
   import { getWorktreesOptions } from '@/generated/api/@tanstack/react-query.gen';
   ```

2. **Thin API Client Wrapper**: 
   - Keep `src/lib/api-client.ts` as a minimal wrapper for data transformation only
   - Never implement business logic in the wrapper
   - Use for UI-specific transformations and type safety

3. **Custom Hooks Pattern**:
   ```typescript
   // ✅ Wrap generated hooks for additional functionality
   export function useWorktrees(options?: UseWorktreesOptions) {
     const query = useQuery(getWorktreesOptions(options));
     const deleteMutation = useMutation({
       mutationFn: async (id: string) => {
         const response = await client.DELETE('/worktrees/{id}', { path: { id } });
         // Handle errors and transform data
       },
     });
     
     return {
       worktrees: query.data?.worktrees || [],
       isLoading: query.isLoading,
       deleteWorktree: deleteMutation.mutate,
     };
   }
   ```

4. **Error Handling**: Let React Query handle errors naturally, display them in UI components
5. **Caching**: Rely on React Query's automatic caching and invalidation
6. **Mutations**: Always invalidate related queries after successful mutations
7. **Legacy API**: Use `src/lib/legacy-api.ts` only for endpoints not yet in OpenAPI spec

#### Forbidden Patterns
- ❌ Manual state management for server data
- ❌ Direct fetch calls in components  
- ❌ Thick API wrapper classes with business logic
- ❌ Manual loading/error state management for server data

### Theme Guidelines
- **CSS Variables**: Theme colors are defined in `src/index.css` using oklch color space
  - Light mode: `:root` selector
  - Dark mode: `.dark` class selector
- **Color Values**: When modifying theme colors, remember that oklch lightness ranges from 0 (black) to 1 (white)
- **Theme-aware Classes**: Always use Tailwind's theme-aware utilities instead of hardcoded colors:
  - ✅ Good: `bg-background`, `text-foreground`, `border-input`
  - ❌ Bad: `bg-slate-900`, `text-white`, `border-slate-700`
- **Dark Mode Classes**: Use Tailwind's `dark:` prefix for dark mode specific styles
  - Example: `bg-black/50 dark:bg-black/90`
- **Component Variants**: When using shadcn/ui components, rely on their built-in variants rather than overriding with custom color classes
- **Testing**: Always test components in both light and dark modes to ensure proper contrast and readability


## Testing

Use `bun test` to run tests.

```ts#index.test.ts
import { test, expect } from "bun:test";

test("hello world", () => {
  expect(1).toBe(1);
});
```

## Frontend

Use HTML imports with `Bun.serve()`. Don't use `vite`. HTML imports fully support React, CSS, Tailwind.

Server:

```ts#index.ts
import index from "./index.html"

Bun.serve({
  routes: {
    "/": index,
    "/api/users/:id": {
      GET: (req) => {
        return new Response(JSON.stringify({ id: req.params.id }));
      },
    },
  },
  // optional websocket support
  websocket: {
    open: (ws) => {
      ws.send("Hello, world!");
    },
    message: (ws, message) => {
      ws.send(message);
    },
    close: (ws) => {
      // handle close
    }
  },
  development: {
    hmr: true,
    console: true,
  }
})
```

HTML files can import .tsx, .jsx or .js files directly and Bun's bundler will transpile & bundle automatically. `<link>` tags can point to stylesheets and Bun's CSS bundler will bundle.

```html#index.html
<html>
  <body>
    <h1>Hello, world!</h1>
    <script type="module" src="./frontend.tsx"></script>
  </body>
</html>
```

With the following `frontend.tsx`:

```tsx#frontend.tsx
import React from "react";

// import .css files directly and it works
import './index.css';

import { createRoot } from "react-dom/client";

const root = createRoot(document.body);

export default function Frontend() {
  return <h1>Hello, world!</h1>;
}

root.render(<Frontend />);
```

Then, run index.ts

```sh
bun --hot ./index.ts
```

For more information, read the Bun API docs in `node_modules/bun-types/docs/**.md`.
