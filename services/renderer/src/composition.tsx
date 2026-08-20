import type { RenderManifest } from "@studio/video-templates";
import React from "react";
import { AbsoluteFill, Audio, Composition, Img, OffthreadVideo, Sequence, interpolate, registerRoot, useCurrentFrame, useVideoConfig } from "remotion";

export type StudioRenderProps = {
  manifest: RenderManifest;
  assetUrls: Record<string, string>;
  qrCodes: Record<string, string>;
};
type ResolvedProduct = { url: string; contentType: string };

const defaultManifest: RenderManifest = {
  renderId: "018f47a0-7b5f-7d5f-9d2a-c5939813086f",
  manifestVersion: 1,
  workspaceId: "018f47a0-7b60-7e88-b3e7-e48888855073",
  campaignId: "018f47a0-7b61-7349-a334-c4a837951586",
  videoProjectId: "018f47a0-7b62-7e8b-b385-3899d9735865",
  videoProjectVersion: 1,
  videoProjectHash: "a".repeat(64),
  language: "vi",
  output: { width: 1080, height: 1920, fps: 30, durationSeconds: 30, codec: "h264" },
  scenes: [{ sceneId: "018f47a0-7b63-7f0b-b2a1-190eb96d84b9", sceneVersion: 1, source: { objectKey: "placeholder.mp4", sha256: "b".repeat(64), contentType: "video/mp4" }, durationMs: 30_000, trimStartMs: 0, trimEndMs: 30_000, muted: true, transition: "cut", productMedia: [] }],
  overlays: [], captions: [], burnCaptions: true, logo: null, music: null, soundEffects: [], musicGainDb: -18, dialogueDuckingDb: -9,
  outputObjectKey: "output.mp4", thumbnailObjectKey: "thumbnail.jpg", createdAt: "2026-08-20T00:00:00Z",
};

const gain = (decibels: number) => Math.pow(10, decibels / 20);

function SceneClip({ scene, source, products }: { scene: RenderManifest["scenes"][number]; source: string; products: ResolvedProduct[] }) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const durationFrames = Math.round(scene.durationMs / 1000 * fps);
  const edge = Math.min(10, Math.floor(durationFrames / 3));
  const opacity = scene.transition === "cut" ? 1 : Math.min(interpolate(frame, [0, edge], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }), interpolate(frame, [durationFrames - edge, durationFrames], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }));
  return <AbsoluteFill style={{ backgroundColor: "#111827", opacity }}><OffthreadVideo src={source} muted={scene.muted} startFrom={Math.round(scene.trimStartMs / 1000 * fps)} endAt={Math.round(scene.trimEndMs / 1000 * fps)} style={{ width: "100%", height: "100%", objectFit: "cover" }} />{products.map((product, index) => product.contentType.startsWith("video/") ? <React.Fragment key={product.url}><OffthreadVideo src={product.url} muted style={{ position: "absolute", right: 56, top: 160 + index * 280, width: 300, height: 240, objectFit: "contain", borderRadius: 24 }} /></React.Fragment> : <Img key={product.url} src={product.url} style={{ position: "absolute", right: 56, top: 160 + index * 280, width: 300, height: 240, objectFit: "contain" }} />)}</AbsoluteFill>;
}

function OverlayLayer({ props }: { props: StudioRenderProps }) {
  const frame = useCurrentFrame();
  return <AbsoluteFill>{props.manifest.overlays.filter((overlay) => frame >= overlay.startFrame && frame < overlay.endFrame).map((overlay, index) => {
    const positions: Record<string, number> = { disclaimer: 70, qr_code: 170, phone: 210, website: 270, cta: 350, lower_third: 420, discount_code: 520, price: 650, headline: 760, logo: 0 };
    const bottom = positions[overlay.type] ?? (overlay.safeZone === "bottom" ? 190 : overlay.safeZone === "action" ? 420 : 700);
    if (overlay.type === "logo" && props.manifest.logo) return <Img key={`${overlay.type}-${index}`} src={props.assetUrls[props.manifest.logo.objectKey]!} style={{ position: "absolute", left: 56, top: 72, width: 180, height: 120, objectFit: "contain" }} />;
    if (overlay.type === "qr_code") return <Img key={`${overlay.type}-${index}`} src={props.qrCodes[overlay.value]!} style={{ position: "absolute", right: 56, bottom, width: 190, height: 190, padding: 12, background: "white", borderRadius: 18 }} />;
    const prominent = overlay.type === "headline" || overlay.type === "price" || overlay.type === "cta";
    return <div key={`${overlay.type}-${index}`} style={{ position: "absolute", left: 56, right: overlay.type === "disclaimer" ? 56 : 220, bottom, color: "white", fontFamily: "Arial, 'Noto Sans', sans-serif", fontSize: prominent ? 58 : overlay.type === "disclaimer" ? 26 : 38, fontWeight: prominent ? 800 : 600, lineHeight: 1.15, padding: prominent ? "20px 28px" : "12px 20px", background: "rgba(10,20,18,0.76)", borderRadius: 20, textShadow: "0 2px 8px rgba(0,0,0,.5)" }}>{overlay.value}</div>;
  })}</AbsoluteFill>;
}

function CaptionLayer({ manifest }: { manifest: RenderManifest }) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const now = frame / fps * 1000;
  const cue = manifest.captions.find((item) => now >= item.startMs && now < item.endMs);
  if (!cue) return null;
  const commerceOverlayVisible = manifest.overlays.some((overlay) => frame >= overlay.startFrame && frame < overlay.endFrame && ["price", "discount_code", "cta", "website", "phone", "qr_code", "disclaimer"].includes(overlay.type));
  return <div style={{ position: "absolute", left: 90, right: 90, bottom: commerceOverlayVisible ? 800 : 120, textAlign: "center", color: "white", fontFamily: "Arial, 'Noto Sans', sans-serif", fontWeight: 800, fontSize: 44, lineHeight: 1.25, padding: "18px 26px", borderRadius: 20, background: "rgba(0,0,0,.72)", textShadow: "0 2px 6px black" }}>{cue.text}</div>;
}

export function StudioFinalVideo(props: StudioRenderProps) {
  const { fps } = useVideoConfig();
  let from = 0;
  return <AbsoluteFill style={{ backgroundColor: "#101714" }}>{props.manifest.scenes.map((scene) => {
    const duration = Math.round(scene.durationMs / 1000 * fps);
    const start = from; from += duration;
    return <Sequence key={scene.sceneId} from={start} durationInFrames={duration}><SceneClip scene={scene} source={props.assetUrls[scene.source.objectKey]!} products={scene.productMedia.map((asset) => ({ url: props.assetUrls[asset.objectKey]!, contentType: asset.contentType }))} /></Sequence>;
  })}{props.manifest.music ? <Audio src={props.assetUrls[props.manifest.music.objectKey]!} volume={(frame) => { const now = frame / fps * 1000; const dialogue = props.manifest.captions.some((cue) => now >= cue.startMs && now < cue.endMs); return gain(props.manifest.musicGainDb + (dialogue ? props.manifest.dialogueDuckingDb : 0)); }} loop /> : null}{props.manifest.soundEffects.map((effect) => <Sequence key={`${effect.source.objectKey}-${effect.startMs}`} from={Math.round(effect.startMs / 1000 * fps)}><Audio src={props.assetUrls[effect.source.objectKey]!} volume={gain(effect.gainDb)} /></Sequence>)}<OverlayLayer props={props} />{props.manifest.burnCaptions ? <CaptionLayer manifest={props.manifest} /> : null}</AbsoluteFill>;
}

function RemotionRoot() {
  return <Composition id="StudioFinalVideo" component={StudioFinalVideo} width={1080} height={1920} fps={30} durationInFrames={900} defaultProps={{ manifest: defaultManifest, assetUrls: {}, qrCodes: {} }} calculateMetadata={({ props }) => ({ durationInFrames: props.manifest.output.durationSeconds * props.manifest.output.fps, width: props.manifest.output.width, height: props.manifest.output.height, fps: props.manifest.output.fps })} />;
}

registerRoot(RemotionRoot);
