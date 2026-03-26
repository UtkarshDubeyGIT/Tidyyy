This is a Tidyyy landing page project built with Next.js, TypeScript, Tailwind CSS, and a shadcn-style UI component setup.

## Getting Started

Run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) to view the landing page.

Core files:

- `src/app/page.tsx` contains the landing page.
- `src/components/ui/*` contains reusable shadcn-style UI primitives (`button`, `card`, `badge`, `accordion`).
- `src/app/globals.css` contains design tokens and shared visual utilities.
- `components.json` stores shadcn configuration metadata.

Build and lint:

```bash
npm run lint
npm run build
```

## Notes

- The UI layer is intentionally shaped like shadcn components and ready for expansion.
- If you want official shadcn CLI-generated components with Radix dependencies, run `npx shadcn@latest init` and `npx shadcn@latest add ...` in this project.
