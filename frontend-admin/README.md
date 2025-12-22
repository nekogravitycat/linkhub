# LinkHub Admin Dashboard

This is the frontend admin dashboard for the LinkHub URL Shortener service. It allows authenticated users to manage short links (create, edit, delete, list).

## Technology Stack

- **Framework**: [Vue 3](https://vuejs.org/) (Composition API)
- **Language**: [TypeScript](https://www.typescriptlang.org/)
- **State Management**: [Pinia](https://pinia.vuejs.org/)
- **Styling**: [Tailwind CSS](https://tailwindcss.com/)
- **UI Components**: [shadcn/vue](https://www.shadcn-vue.com/)
- **HTTP Client**: [Axios](https://axios-http.com/)
- **Build Tool**: [Vite](https://vitejs.dev/)

## Project Structure

```text
frontend-admin/
├── src/
│   ├── components/         # Vue components
│   │   ├── ui/             # Reusable shadcn/vue UI components
│   │   └── LinkDialog.vue  # Dialog for creating/editing links
│   ├── lib/
│   │   └── api.ts          # Axios instance with Cloudflare Access credentials config
│   ├── router/
│   │   └── index.ts        # Vue Router configuration
│   ├── stores/
│   │   └── links.ts        # Pinia store for Link state management
│   ├── views/
│   │   └── HomeView.vue    # Main dashboard view
│   ├── App.vue             # Root component (handles dark mode)
│   ├── main.ts             # Application entry point
│   └── style.css           # Global styles and Tailwind imports
├── .gitignore
├── component.json          # shadcn/vue configuration
├── env.d.ts
├── index.html
├── package.json
├── pnpm-lock.yaml
├── tsconfig.json
├── vite.config.ts          # Vite configuration
└── openapi.yml             # API specification
```

## Setup & Installation

1.  **Install dependencies**:

    ```bash
    pnpm install
    ```

2.  **Install shadcn components** (if needed):
    The project uses `shadcn-vue`. If you need to add new components:
    ```bash
    pnpm dlx shadcn-vue@latest add [component-name]
    ```

## Development

1.  **Start the development server**:

    ```bash
    pnpm dev
    ```

2.  **Linting & Formatting**:
    ```bash
    pnpm lint
    pnpm format
    ```

## Key Features & implementation Details

### Authentication

The application relies on Cloudflare Access for authentication. No login page is implemented within the app.

- **Session**: Managed via cookies.
- **API Client**: configured in `src/lib/api.ts` with `withCredentials: true` to ensure cookies are sent with every request.

### State Management

`src/stores/links.ts` manages the application state:

- `links`: Array of link objects.
- `loading`: Boolean loading state.
- `error`: Error messages.
- Actions: `fetchLinks`, `createLink`, `updateLink`, `deleteLink`.

### Styling

- **Dark Mode**: Enforced by default in `App.vue` (`document.documentElement.classList.add('dark')`).
- **Responsive**: `HomeView.vue` features a responsive table that hides less important columns on mobile devices.

## Scripts Flow & Application Logic

### 1. View Layer (Components & Views)

- **Implementation**: `src/views/HomeView.vue`, `src/components/LinkDialog.vue`
- **Role**: Handles user interactions and display logic.
- **Flow**:
  - Components use `useLinksStore()` to access state (`links`, `loading`) and dispatch actions (`createLink`, `deleteLink`).
  - They do **not** make API calls directly.
  - Example: When "Create Link" is clicked, `LinkDialog.vue` calls `store.createLink(slug, url)`.

### 2. Store Layer (Pinia)

- **Implementation**: `src/stores/links.ts`
- **Role**: Manages application state and business logic.
- **Flow**:
  - **State**: Holds the list of links (`links`), loading status (`loading`), and errors (`error`).
  - **Actions**:
    - `fetchLinks()`: Calls API to get links -> updates `links` state.
    - `createLink()`: Calls API to post data -> triggers `fetchLinks()` to refresh list.
    - `updateLink()`: Calls API to patch data -> triggers `fetchLinks()`.
    - `deleteLink()`: Calls API into delete -> triggers `fetchLinks()`.

### 3. API Layer (Axios)

- **Implementation**: `src/lib/api.ts`
- **Role**: handles raw HTTP requests and configuration.
- **Flow**:
  - Exports an `axios` instance configured with `baseURL` and `withCredentials: true`.
  - This ensures all requests automatically include the Cloudflare Access cookies required for authentication.
