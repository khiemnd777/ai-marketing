"use client";

import { Check } from "lucide-react";
import type { FormOption } from "@/lib/form-options";

export function OptionPicker({ options, value, onChange, label, disabled = false, multiple = true, emptyText = "Chưa có lựa chọn phù hợp." }: { options: readonly FormOption[]; value: string[]; onChange: (values: string[]) => void; label: string; disabled?: boolean; multiple?: boolean; emptyText?: string }) {
  const toggle = (id: string) => {
    if (multiple) onChange(value.includes(id) ? value.filter((current) => current !== id) : [...value, id]);
    else onChange(value.includes(id) ? [] : [id]);
  };
  return <fieldset disabled={disabled} className="min-w-0"><legend className="mb-2 text-sm font-semibold text-[var(--ink)]">{label}</legend>{options.length === 0 ? <div className="rounded-xl border border-dashed border-[var(--line)] p-3 text-xs text-[var(--muted)]">{emptyText}</div> : <div className="grid max-h-60 gap-2 overflow-auto rounded-xl border border-[var(--line)] bg-white p-2 sm:grid-cols-2">{options.map((option) => { const selected = value.includes(option.value); return <button key={option.value} type="button" role={multiple ? "checkbox" : "radio"} aria-checked={selected} disabled={disabled} className={`flex min-h-11 items-center gap-3 rounded-xl border px-3 py-2 text-left text-sm transition ${selected ? "border-[var(--moss)] bg-[#edf3e6]" : "border-transparent hover:border-[var(--line)] hover:bg-[#f7f8f3]"} disabled:cursor-not-allowed disabled:opacity-60`} onClick={() => toggle(option.value)}><span className={`grid size-7 shrink-0 place-items-center rounded-lg ${selected ? "bg-[var(--moss)] text-white" : "bg-[#edf0e7]"}`}>{selected ? <Check className="size-4" /> : null}</span><span className="min-w-0 flex-1 font-semibold">{option.label}</span></button>; })}</div>}</fieldset>;
}
