# ADR-008: Remotion Licensing and Isolated Renderer

- Status: Accepted
- Date: 2026-08-20

## Context

Final outputs combine generated clips, real product media, exact structured facts, captions, branding, music, and transitions. Rendering requires a Node/media runtime and has materially different CPU and temporary-storage behavior from the Go API.

## Decision

Use Remotion 4 in a separate Node.js 24 renderer service. Pin every `remotion` and `@remotion/*` package to the same exact version and verify the organization’s use against Remotion’s current license before production deployment. Use FFmpeg/ffprobe for encoding inspection and Sharp for raster processing. Authenticate manifests, validate input hashes, and clean per-render temporary directories.

## Consequences

Rendering can scale and fail independently without contaminating API memory or process health. License verification is an operational release gate rather than an implicit assumption.
