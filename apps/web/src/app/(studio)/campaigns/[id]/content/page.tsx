"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Save } from "lucide-react";
import { Suspense, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { CampaignHeader, GenerationPanel, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, SkeletonRows, StatePanel, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

type Variant = components["schemas"]["CampaignContentVariant"];
export default function ContentPage() { return <Suspense fallback={<SkeletonRows />}><Content /></Suspense>; }

function Content() {
  const scope = useCampaignRoute();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: ["campaign-content", scope.campaignId], enabled: !!scope.clientId && !!scope.workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/content", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải nội dung."); return data; } });
  const refresh = () => qc.invalidateQueries({ queryKey: ["campaign-content", scope.campaignId] });
  return <CampaignHeader active="/content" title="AI Content" description="Mười bốn biến thể bắt buộc được kiểm tra chung với Product Truth. Sửa một biến thể tạo version mới và vô hiệu hóa approval cũ."><GenerationPanel operation="CONTENT" />{query.isLoading ? <SkeletonRows /> : query.error ? <StatePanel title="Không thể tải nội dung" tone="danger">{query.error.message}</StatePanel> : query.data?.items.length === 0 ? <StatePanel title="Chưa có nội dung">Khóa một concept rồi chạy AI content để tạo đủ 14 biến thể.</StatePanel> : <div className="grid gap-4 lg:grid-cols-2">{query.data?.items.map((item) => <VariantCard key={item.id} item={item} scope={scope} onChanged={refresh} />)}</div>}</CampaignHeader>;
}

function VariantCard({ item, scope, onChanged }: { item: Variant; scope: ReturnType<typeof useCampaignRoute>; onChanged: () => Promise<unknown> }) {
  const { canOperate, canReview } = usePermissions();
  const [content, setContent] = useState(item.content);
  const update = useMutation({ mutationFn: async () => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/content/{contentId}", { params: { path: { ...scope, contentId: item.id } }, body: { content, version: item.version } }); if (error || !data) throw apiError(error, "Nội dung vi phạm Product Truth hoặc đã thay đổi."); return data; }, onSuccess: onChanged });
  const approve = useMutation({ mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/content/{contentId}/approve", { params: { path: { ...scope, contentId: item.id } }, body: { version: item.version, notes: "Đã review nội dung và Product Truth" } }); if (error || !data) throw apiError(error, "Không thể duyệt nội dung."); return data; }, onSuccess: onChanged });
  return <Card className="p-5"><div className="mb-3 flex items-center gap-2"><h2 className="font-serif text-lg font-bold">{item.key.replaceAll("_", " ")}</h2><Badge>{item.platform}</Badge><Badge tone={item.status === "APPROVED" ? "good" : "warn"}>{item.status}</Badge><span className="ml-auto text-xs text-[var(--muted)]">v{item.currentVersion}</span></div><textarea aria-label={item.key} className={textareaClass} disabled={!canOperate} value={content} onChange={(e) => setContent(e.target.value)} />{(update.error || approve.error) ? <p className="mt-3 text-sm text-[var(--coral)]">{update.error?.message ?? approve.error?.message}</p> : null}<div className="mt-4 flex gap-2">{canOperate ? <Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" disabled={update.isPending || content.trim() === item.content} onClick={() => update.mutate()}><Save className="mr-2 size-4" />Lưu version</Button> : null}{canReview && item.status !== "APPROVED" ? <Button disabled={approve.isPending} onClick={() => approve.mutate()}><Check className="mr-2 size-4" />Duyệt</Button> : null}</div></Card>;
}
