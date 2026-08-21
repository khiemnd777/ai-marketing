"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Save } from "lucide-react";
import Link from "next/link";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { BrandLogoPanel } from "@/components/brand-logo-panel";
import { useScopedEntityId, useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";
import { studioRoutes } from "@/lib/studio-routes";

type BrandInput = components["schemas"]["BrandProfileInput"];
type Brand = components["schemas"]["BrandProfile"];

export default function BrandPage() {
  return <Suspense fallback={<SkeletonRows />}><BrandContent /></Suspense>;
}

function BrandContent() {
  const id = useScopedEntityId("brandId");
  const { clientId, workspaceId } = useStudioScope();
  const brand = useQuery({
    queryKey: ["brand", clientId, workspaceId, id],
    enabled: Boolean(clientId && workspaceId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/brands/{brandId}", { params: { path: { clientId, workspaceId, brandId: id } } });
      if (error || !data) throw apiError(error, "Không thể tải thương hiệu.");
      return data;
    },
  });
  if (!clientId || !workspaceId) return <StatePanel title="Thiếu phạm vi workspace">Mở hồ sơ từ trang workspace.</StatePanel>;
  if (brand.isLoading) return <SkeletonRows />;
  if (brand.error) return <StatePanel title="Không thể tải thương hiệu" tone="danger">{brand.error.message}</StatePanel>;
  return <BrandEditor key={brand.data!.version} data={brand.data!} clientId={clientId} workspaceId={workspaceId} />;
}

function BrandEditor({ data, clientId, workspaceId }: { data: Brand; clientId: string; workspaceId: string }) {
  const [form, setForm] = useState<BrandInput>(() => brandInput(data));
  const [logoEligible, setLogoEligible] = useState((data.logoAssetIds ?? []).length === 0);
  const { canOperate } = usePermissions();
  const qc = useQueryClient();
  const scope = useMemo(() => ({ clientId, workspaceId }), [clientId, workspaceId]);
  const dirty = useMemo(() => JSON.stringify(form) !== JSON.stringify(brandInput(data)), [data, form]);
  const save = useMutation({
    mutationFn: async () => {
      const { data: updated, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/brands/{brandId}", { params: { path: { clientId, workspaceId, brandId: data.id } }, body: { ...form, version: data.version, changeSummary: "Cập nhật hồ sơ và logo từ Studio" } });
      if (error || !updated) throw apiError(error, "Không thể lưu hồ sơ. Logo phải được duyệt, còn hạn và xử lý hoàn tất.");
      return updated;
    },
    onSuccess: async () => Promise.all([
      qc.invalidateQueries({ queryKey: ["brand", clientId, workspaceId, data.id] }),
      qc.invalidateQueries({ queryKey: ["brands", clientId, workspaceId] }),
      qc.invalidateQueries({ queryKey: ["media", clientId, workspaceId] }),
    ]),
  });
  const set = useCallback((key: keyof BrandInput, value: unknown) => setForm((current) => ({ ...current, [key]: value })), []);
  const setLogoEligibility = useCallback((valid: boolean) => setLogoEligible(valid), []);

  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => {
      if (!dirty) return;
      event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  return <>
    <Link href={studioRoutes.brands(clientId, workspaceId)} onClick={(event) => { if (dirty && !window.confirm("Rời trang và bỏ các thay đổi chưa lưu?")) event.preventDefault(); }} className="mb-5 inline-flex min-h-11 items-center gap-2 text-sm font-bold text-[var(--moss)]"><ArrowLeft className="size-4" />Thương hiệu</Link>
    <PageHeader eyebrow={`Hồ sơ v${data.currentVersion}`} title={data.name} description="Mỗi lần lưu tạo một phiên bản bất biến. Nội dung AI và video về sau dùng đúng phiên bản được chọn." action={<div className="flex items-center gap-3"><Badge tone="good">{data.status}</Badge>{canOperate ? <Button disabled={save.isPending || !logoEligible || !dirty} onClick={() => save.mutate()}><Save className="mr-2 size-4" />{save.isPending ? "Đang lưu…" : "Lưu phiên bản"}</Button> : null}</div>} />
    <BrandLogoPanel brandId={data.id} scope={scope} value={form.logoAssetIds ?? []} onChange={(ids) => set("logoAssetIds", ids)} onEligibilityChange={setLogoEligibility} />
    <Card className="p-6"><fieldset disabled={!canOperate} className="grid gap-5 md:grid-cols-2"><Field label="Tên thương hiệu"><input className={inputClass} value={form.name} onChange={(event) => set("name", event.target.value)} /></Field><Field label="Ngôn ngữ chính"><select className={inputClass} value={form.primaryLanguage} onChange={(event) => set("primaryLanguage", event.target.value as "vi" | "en")}><option value="vi">Tiếng Việt</option><option value="en">English</option></select></Field><Color disabled={!canOperate} label="Màu chính" value={form.primaryColor} onChange={(value) => set("primaryColor", value)} /><Color disabled={!canOperate} label="Màu phụ" value={form.secondaryColor} onChange={(value) => set("secondaryColor", value)} /><div className="md:col-span-2"><Field label="Giọng điệu"><textarea className={textareaClass} value={form.toneOfVoice ?? ""} onChange={(event) => set("toneOfVoice", event.target.value)} /></Field></div><div className="md:col-span-2"><Field label="Đối tượng mục tiêu"><textarea className={textareaClass} value={form.targetAudience ?? ""} onChange={(event) => set("targetAudience", event.target.value)} /></Field></div><Field label="Thông điệp chính"><input className={inputClass} value={form.mainMessage ?? ""} onChange={(event) => set("mainMessage", event.target.value)} /></Field><Field label="CTA mặc định"><input className={inputClass} value={form.defaultCta ?? ""} onChange={(event) => set("defaultCta", event.target.value)} /></Field><Field label="Website"><input className={inputClass} type="url" value={form.website ?? ""} onChange={(event) => set("website", event.target.value || null)} /></Field><Field label="Điện thoại"><input className={inputClass} value={form.phoneNumber ?? ""} onChange={(event) => set("phoneNumber", event.target.value || null)} /></Field><Field label="Thuật ngữ ưu tiên (mỗi dòng)"><textarea className={textareaClass} value={(form.preferredTerminology ?? []).join("\n")} onChange={(event) => set("preferredTerminology", event.target.value.split("\n").map((value) => value.trim()).filter(Boolean))} /></Field><Field label="Thuật ngữ cấm (mỗi dòng)"><textarea className={textareaClass} value={(form.prohibitedTerminology ?? []).join("\n")} onChange={(event) => set("prohibitedTerminology", event.target.value.split("\n").map((value) => value.trim()).filter(Boolean))} /></Field><div className="md:col-span-2"><Field label="Tuyên bố miễn trừ"><textarea className={textareaClass} value={form.defaultDisclaimer ?? ""} onChange={(event) => set("defaultDisclaimer", event.target.value)} /></Field></div></fieldset>{!canOperate ? <p className="mt-4 text-sm text-[var(--muted)]">Bạn có thể xem hồ sơ và duyệt media theo quyền được cấp; chỉ Admin/Operator được lưu phiên bản Brand.</p> : null}{save.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{save.error.message}</p> : null}</Card>
  </>;
}

function Color({ label, value, onChange, disabled }: { label: string; value?: string | null; onChange: (value: string | null) => void; disabled: boolean }) {
  return <Field label={label}><div className="flex gap-3"><input disabled={disabled} aria-label={`${label} picker`} className="size-11 rounded-xl border border-[var(--line)] bg-white p-1" type="color" value={value ?? "#26664f"} onChange={(event) => onChange(event.target.value)} /><input disabled={disabled} className={inputClass} value={value ?? ""} placeholder="#26664f" onChange={(event) => onChange(event.target.value || null)} /></div></Field>;
}

function brandInput(data: Brand): BrandInput {
  return {
    name: data.name,
    logoAssetIds: data.logoAssetIds,
    primaryColor: data.primaryColor,
    secondaryColor: data.secondaryColor,
    backgroundColor: data.backgroundColor,
    headingFont: data.headingFont,
    bodyFont: data.bodyFont,
    toneOfVoice: data.toneOfVoice,
    primaryLanguage: data.primaryLanguage,
    targetAudience: data.targetAudience,
    mainMessage: data.mainMessage,
    defaultCta: data.defaultCta,
    website: data.website,
    phoneNumber: data.phoneNumber,
    preferredTerminology: data.preferredTerminology,
    prohibitedTerminology: data.prohibitedTerminology,
    defaultDisclaimer: data.defaultDisclaimer,
    defaultVideoStyle: data.defaultVideoStyle,
    defaultMusicStyle: data.defaultMusicStyle,
    changeSummary: data.changeSummary,
    version: data.version,
  };
}
