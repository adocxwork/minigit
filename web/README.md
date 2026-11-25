# MiniGit dashboard

The React and TypeScript dashboard uses the Go API served by `mgit ui`. See the [build instructions](../README.md) and the [API and storage walkthrough](../PROJECT_DOCUMENTATION.md).

Run `npm ci`, then `npm run lint` or `npm run build`. For development with hot reload, start a UI-enabled MiniGit server on port 8080 and run `npm run dev` in this directory. Vite proxies `/api` to the server.
