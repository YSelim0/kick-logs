# Build stage: install the full toolchain and produce a production build.
FROM node:20-alpine AS build

ENV NEXT_TELEMETRY_DISABLED=1
ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH

WORKDIR /repo

RUN corepack enable && corepack prepare pnpm@8.11.0 --activate

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json

RUN pnpm install --frozen-lockfile --filter @kick-logs/web...

COPY apps/web apps/web

# NEXT_PUBLIC_* values are inlined into the client bundle at build time, so the
# API base URL must be present during `pnpm build`, not only at runtime.
ARG NEXT_PUBLIC_API_BASE_URL=http://localhost:8000
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL

WORKDIR /repo/apps/web
RUN pnpm build

# Runtime stage: production dependencies only + the compiled build artifacts.
FROM node:20-alpine AS runtime

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH

WORKDIR /repo

RUN corepack enable && corepack prepare pnpm@8.11.0 --activate

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json

RUN pnpm install --prod --frozen-lockfile --filter @kick-logs/web...

COPY --from=build /repo/apps/web/.next apps/web/.next
COPY --from=build /repo/apps/web/public apps/web/public
COPY --from=build /repo/apps/web/next.config.mjs apps/web/next.config.mjs

WORKDIR /repo/apps/web

EXPOSE 3000

CMD ["pnpm", "start"]
