import type { RenderManifest } from "@studio/video-templates";

const overlayBottoms: Record<string, number> = {
  disclaimer: 70,
  qr_code: 170,
  phone: 210,
  website: 270,
  cta: 350,
  lower_third: 420,
  discount_code: 520,
  price: 650,
  headline: 760,
  logo: 0,
};

const commerceOverlayTypes = new Set(["price", "discount_code", "cta", "website", "phone", "qr_code", "disclaimer"]);

export function decibelGain(decibels: number) {
  return Math.pow(10, decibels / 20);
}

export function overlayPlacement(type: string, safeZone: "title" | "action" | "bottom") {
  const prominent = type === "headline" || type === "price" || type === "cta";
  return {
    bottom: overlayBottoms[type] ?? (safeZone === "bottom" ? 190 : safeZone === "action" ? 420 : 700),
    fontSize: prominent ? 58 : type === "disclaimer" ? 26 : 38,
    prominent,
  };
}

export function captionBottom(manifest: RenderManifest, frame: number) {
  return manifest.overlays.some((overlay) => frame >= overlay.startFrame && frame < overlay.endFrame && commerceOverlayTypes.has(overlay.type)) ? 800 : 120;
}

export function buildRenderPlan(manifest: RenderManifest) {
  let from = 0;
  const scenes = manifest.scenes.map((scene) => {
    const durationInFrames = Math.round(scene.durationMs / 1000 * manifest.output.fps);
    const planned = { sceneId: scene.sceneId, from, durationInFrames, trimStartFrame: Math.round(scene.trimStartMs / 1000 * manifest.output.fps), trimEndFrame: Math.round(scene.trimEndMs / 1000 * manifest.output.fps), transition: scene.transition };
    from += durationInFrames;
    return planned;
  });
  return {
    durationInFrames: manifest.output.durationSeconds * manifest.output.fps,
    scenes,
    overlays: manifest.overlays.map((overlay) => ({ type: overlay.type, startFrame: overlay.startFrame, endFrame: overlay.endFrame, ...overlayPlacement(overlay.type, overlay.safeZone) })),
    captions: manifest.captions.map((cue) => ({ text: cue.text, startFrame: Math.round(cue.startMs / 1000 * manifest.output.fps), endFrame: Math.round(cue.endMs / 1000 * manifest.output.fps) })),
    audio: { musicGain: decibelGain(manifest.musicGainDb), duckedMusicGain: decibelGain(manifest.musicGainDb + manifest.dialogueDuckingDb) },
  };
}
