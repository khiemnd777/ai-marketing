"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Ban, Check, ExternalLink, Send } from "lucide-react";
import Link from "next/link";
import { Suspense, useState } from "react";
import { CampaignHeader, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type SocialPost = components["schemas"]["SocialPost"];

function PublishingContent() {
  const scope = useCampaignRoute();
  const queryClient = useQueryClient();
  const [accountId, setAccountId] = useState("");
  const [mediaAssetId, setMediaAssetId] = useState("");
  const [caption, setCaption] = useState("Khám phá sản phẩm và tìm hiểu thêm ngay hôm nay.");
  const [scheduledAt, setScheduledAt] = useState("");
  const connection = useQuery({ queryKey: ["meta-connection", scope.clientId, scope.workspaceId], enabled: !!scope.clientId && !!scope.workspaceId, retry: false, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/meta/connection", { params: { path: scope } }); if (error || !data) return null; return data; } });
  const renders = useQuery({ queryKey: ["final-renders", scope], enabled: !!scope.clientId && !!scope.workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải final render."); return data; } });
  const posts = useQuery({ queryKey: ["social-posts", scope], enabled: !!scope.clientId && !!scope.workspaceId, refetchInterval: (query) => query.state.data?.items.some((item) => ["APPROVED", "SCHEDULED", "PUBLISHING"].includes(item.status)) ? 2500 : false, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/social-posts", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải publishing status."); return data; } });
  const selectedAsset = renders.data?.items.find((item) => item.selected)?.outputAssetId ?? "";
  const chosenAccount = accountId || connection.data?.accounts.find((item) => item.status === "CONNECTED")?.id || "";
  const chosenAsset = mediaAssetId || selectedAsset || "";
  const refresh = async () => queryClient.invalidateQueries({ queryKey: ["social-posts", scope] });
  const create = useMutation({ mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/social-posts", { params: { path: scope, header: { "Idempotency-Key": newIdempotencyKey() } }, body: { socialAccountId: chosenAccount, mediaAssetId: chosenAsset, caption, scheduledAt: scheduledAt ? new Date(scheduledAt).toISOString() : null } }); if (error || !data) throw apiError(error, "Chỉ final render đã duyệt và Meta account đang kết nối mới được xuất bản."); return data; }, onSuccess: async () => { setScheduledAt(""); await refresh(); } });
  return <CampaignHeader active="/publishing" title="Xuất bản Meta" description="Mỗi Page hoặc Instagram Business có caption, lịch và approval riêng. Worker chỉ publish hash đã được duyệt.">
    {!connection.data ? <StatePanel title="Chưa kết nối Meta">Kết nối tại <Link className="font-bold text-[var(--moss)] underline" href={`/settings/meta?clientId=${scope.clientId}&workspaceId=${scope.workspaceId}`}>Settings → Meta</Link> trước khi tạo lịch xuất bản.</StatePanel> : <Card className="p-6"><h2 className="font-serif text-2xl font-bold">Tạo publishing request</h2><div className="mt-5 grid gap-4 md:grid-cols-2">
      <Field label="Page / Instagram"><select className={inputClass} value={chosenAccount} onChange={(event) => setAccountId(event.target.value)}>{connection.data.accounts.filter((item) => item.status === "CONNECTED" || item.status === "EXPIRING").map((item) => <option key={item.id} value={item.id}>{item.platform} · {item.name}</option>)}</select></Field>
      <Field label="Final media asset"><input className={inputClass} value={chosenAsset} onChange={(event) => setMediaAssetId(event.target.value)} placeholder="UUID của final media asset" /></Field>
      <Field label="Lịch (để trống = ngay sau duyệt)"><input className={inputClass} type="datetime-local" value={scheduledAt} onChange={(event) => setScheduledAt(event.target.value)} /></Field>
      <div className="md:row-span-2"><Field label="Caption riêng cho platform"><textarea className={textareaClass} value={caption} maxLength={2200} onChange={(event) => setCaption(event.target.value)} /></Field><p className="mt-1 text-right text-xs text-[var(--muted)]">{caption.length}/2200</p></div>
    </div>{create.error ? <p className="mt-3 text-sm text-[var(--coral)]">{create.error.message}</p> : null}<Button className="mt-5" disabled={create.isPending || !chosenAccount || !chosenAsset || !caption.trim()} onClick={() => create.mutate()}><Send className="mr-2 size-4" />Gửi để duyệt</Button></Card>}
    <section className="mt-7"><h2 className="mb-4 font-serif text-2xl font-bold">Publishing history</h2>{posts.isLoading ? <SkeletonRows /> : posts.error ? <StatePanel title="Không thể tải publishing" tone="danger">{posts.error.message}</StatePanel> : posts.data?.items.length ? <div className="grid gap-4">{posts.data.items.map((item) => <PostCard key={item.id} item={item} scope={scope} refresh={refresh} />)}</div> : <StatePanel title="Chưa có publishing request">Tạo một request cho từng platform để giữ caption và lịch độc lập.</StatePanel>}</section>
  </CampaignHeader>;
}

function PostCard({ item, scope, refresh }: { item: SocialPost; scope: ReturnType<typeof useCampaignRoute>; refresh: () => Promise<unknown> }) {
  const [notes, setNotes] = useState("Đã kiểm tra caption, media và tài khoản đích.");
  const review = useMutation({ mutationFn: async (action: "APPROVE" | "REJECT") => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/social-posts/{postId}/review", { params: { path: { ...scope, postId: item.id } }, body: { action, version: item.version, notes } }); if (error || !data) throw apiError(error, "Không thể ghi publishing review."); return data; }, onSuccess: refresh });
  const tone = item.status === "PUBLISHED" ? "good" : item.status.includes("FAILURE") || item.status === "FAILED" ? "danger" : item.status === "APPROVAL_REQUIRED" ? "warn" : "neutral";
  return <Card className="p-5"><div className="flex flex-wrap items-center gap-2"><Badge tone={tone}>{item.status}</Badge><Badge>{item.platform}</Badge><span className="text-xs text-[var(--muted)]">{item.scheduledAt ? new Date(item.scheduledAt).toLocaleString("vi-VN") : "Xuất bản ngay sau duyệt"}</span></div><p className="mt-4 whitespace-pre-wrap text-sm leading-6">{item.caption}</p>{item.errorMessage ? <p className="mt-3 text-sm text-[var(--coral)]">{item.errorCode}: {item.errorMessage}</p> : null}{item.status === "APPROVAL_REQUIRED" ? <Field label="Ghi chú reviewer"><textarea className={`${textareaClass} mt-4`} value={notes} onChange={(event) => setNotes(event.target.value)} /></Field> : null}{review.error ? <p className="mt-3 text-sm text-[var(--coral)]">{review.error.message}</p> : null}<div className="mt-4 flex flex-wrap gap-2">{item.status === "APPROVAL_REQUIRED" ? <><Button disabled={review.isPending} onClick={() => review.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt publish</Button><Button className="bg-[var(--coral)]" disabled={review.isPending || !notes.trim()} onClick={() => review.mutate("REJECT")}><Ban className="mr-2 size-4" />Từ chối</Button></> : null}{item.publicUrl ? <a className="inline-flex min-h-10 items-center rounded-full px-4 text-sm font-bold text-[var(--moss)] ring-1 ring-[var(--line)]" href={item.publicUrl} target="_blank" rel="noreferrer"><ExternalLink className="mr-2 size-4" />Mở bài đăng</a> : null}</div></Card>;
}

export default function PublishingPage() {
  return <Suspense fallback={<SkeletonRows />}><PublishingContent /></Suspense>;
}
