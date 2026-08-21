FROM oven/bun:1.3.1 AS dependencies
WORKDIR /workspace
COPY package.json bun.lock bunfig.toml ./
COPY apps/web/package.json apps/web/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/contracts/package.json packages/contracts/package.json
COPY packages/design-tokens/package.json packages/design-tokens/package.json
COPY packages/shared-config/package.json packages/shared-config/package.json
COPY packages/video-templates/package.json packages/video-templates/package.json
COPY services/renderer/package.json services/renderer/package.json
RUN bun install --frozen-lockfile

FROM node:26.7.0-bookworm AS build
WORKDIR /workspace
COPY --from=dependencies /workspace /workspace
COPY apps/web apps/web
COPY packages packages
COPY openapi openapi
COPY verticals verticals
ENV NEXT_TELEMETRY_DISABLED=1
RUN cd apps/web && node node_modules/next/dist/bin/next build --webpack

FROM node:26.7.0-bookworm-slim AS runtime
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
WORKDIR /app
COPY --from=build --chown=node:node /workspace/apps/web/.next/standalone ./
COPY --from=build --chown=node:node /workspace/apps/web/.next/static ./apps/web/.next/static
COPY --from=build --chown=node:node /workspace/apps/web/public ./apps/web/public
USER node
EXPOSE 3000
CMD ["node", "apps/web/server.js"]
