FROM node:20-alpine

ENV NEXT_TELEMETRY_DISABLED=1
ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH

WORKDIR /repo

RUN corepack enable && corepack prepare pnpm@8.11.0 --activate

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json

RUN pnpm install --frozen-lockfile --filter @kick-logs/web...

COPY apps/web apps/web

WORKDIR /repo/apps/web

EXPOSE 3000

CMD ["pnpm", "dev"]
