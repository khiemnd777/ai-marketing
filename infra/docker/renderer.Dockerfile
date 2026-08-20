FROM oven/bun:1.3.14 AS dependencies
WORKDIR /workspace
COPY package.json bun.lock bunfig.toml ./
COPY apps/web/package.json apps/web/package.json
COPY services/renderer/package.json services/renderer/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/video-templates/package.json packages/video-templates/package.json
COPY packages/contracts/package.json packages/contracts/package.json
COPY packages/design-tokens/package.json packages/design-tokens/package.json
COPY packages/shared-config/package.json packages/shared-config/package.json
RUN bun install --frozen-lockfile

FROM dependencies AS build
COPY services/renderer services/renderer
COPY packages packages
RUN bun --filter @studio/video-templates build && bun --filter @studio/renderer build

FROM node:24.6.0-bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends chromium ffmpeg fonts-noto-core ca-certificates \
    && rm -rf /var/lib/apt/lists/*
ENV NODE_ENV=production RENDERER_HOST=0.0.0.0 RENDERER_PORT=8090 REMOTION_BROWSER_EXECUTABLE=/usr/bin/chromium RENDER_TEMP_DIR=/tmp/studio-renderer
WORKDIR /app/services/renderer
COPY --from=build --chown=node:node /workspace/services/renderer/dist ./dist
COPY --from=dependencies --chown=node:node /workspace/node_modules /app/node_modules
COPY --from=dependencies --chown=node:node /workspace/services/renderer/node_modules /app/services/renderer/node_modules
COPY --from=build --chown=node:node /workspace/packages/video-templates /app/packages/video-templates
COPY --from=build --chown=node:node /workspace/packages/contracts /app/packages/contracts
COPY services/renderer/package.json ./package.json
USER node
EXPOSE 8090
CMD ["node", "dist/server.js"]
