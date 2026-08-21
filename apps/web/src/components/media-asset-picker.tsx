"use client";

import type { components } from "@studio/api-client";
import { Check, Film, ImageIcon } from "lucide-react";
import { Badge } from "@/components/ui";

type Asset = components["schemas"]["MediaAsset"];

export function MediaAssetPicker({ assets, value, onChange, label, disabled = false, multiple = true, allowEmpty = true }: { assets: Asset[]; value: string[]; onChange: (ids: string[]) => void; label: string; disabled?: boolean; multiple?: boolean; allowEmpty?: boolean }) {
  const toggle = (id: string) => {
    if (multiple) onChange(value.includes(id) ? value.filter((current) => current !== id) : [...value, id]);
    else onChange(value.includes(id) && allowEmpty ? [] : [id]);
  };
  return <fieldset disabled={disabled} className="min-w-0"><legend className="mb-2 text-xs font-bold text-[var(--muted)]">{label}</legend>{assets.length === 0 ? <div className="rounded-xl border border-dashed border-[var(--line)] p-3 text-xs text-[var(--muted)]">Chưa có asset APPROVED phù hợp với sản phẩm campaign.</div> : <div className="grid max-h-60 gap-2 overflow-auto rounded-xl border border-[var(--line)] bg-white p-2 sm:grid-cols-2">{assets.map((asset) => { const selected = value.includes(asset.id); return <button key={asset.id} type="button" role={multiple ? "checkbox" : "radio"} aria-checked={selected} disabled={disabled} className={`flex min-h-14 items-center gap-3 rounded-xl border p-3 text-left transition ${selected ? "border-[var(--moss)] bg-[#edf3e6]" : "border-transparent hover:border-[var(--line)] hover:bg-[#f7f8f3]"} disabled:cursor-not-allowed disabled:opacity-60`} onClick={() => toggle(asset.id)}><span className={`grid size-8 shrink-0 place-items-center rounded-lg ${selected ? "bg-[var(--moss)] text-white" : "bg-[#edf0e7]"}`}>{selected ? <Check className="size-4" /> : asset.assetType === "VIDEO" || asset.assetType === "SCREEN_RECORDING" ? <Film className="size-4" /> : <ImageIcon className="size-4" />}</span><span className="min-w-0 flex-1"><span className="block truncate text-sm font-bold">{asset.name}</span><span className="mt-1 block truncate text-xs text-[var(--muted)]">{asset.category || "Chưa phân loại"}</span></span><Badge tone="good">APPROVED</Badge></button>; })}</div>}</fieldset>;
}
