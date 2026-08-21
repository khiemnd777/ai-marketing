"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, CircleOff, KeyRound, ShieldCheck, XCircle } from "lucide-react";
import { useStudioScope } from "@/components/studio-scope";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { Badge, Button, Card, Field, inputClass, PageHeader, SkeletonRows, StatePanel } from "@/components/ui";
import { api } from "@/lib/api";
import { providerSettingSuggestions } from "@/lib/form-options";
import { apiError } from "@/lib/problem";

type Provider = components["schemas"]["ProviderConfiguration"];
type ProviderKind = components["schemas"]["ProviderKind"];
type SettingValue = components["schemas"]["ProviderSettingValue"];

type FieldDefinition = {
  name: string;
  label: string;
  type?: "text" | "url" | "number" | "select";
  required?: boolean;
  min?: number;
  step?: string;
  options?: string[];
  help?: string;
};

type ProviderDefinition = {
  title: string;
  description: string;
  fields: FieldDefinition[];
  secrets: Array<{ name: string; label: string }>;
};

const definitions: Record<ProviderKind, ProviderDefinition> = {
  OPENAI: {
    title: "OpenAI",
    description: "Sinh concept, content, script, scene và transcription.",
    fields: [
      { name: "baseUrl", label: "Endpoint", type: "url", required: true },
      { name: "model", label: "Generation model", required: true },
      { name: "transcriptionModel", label: "Transcription model", required: true },
      { name: "reasoningEffort", label: "Reasoning effort", type: "select", options: ["none", "low", "medium", "high", "xhigh"], required: true },
      { name: "timeoutSeconds", label: "Timeout (giây)", type: "number", min: 1, required: true },
      { name: "inputUsdPer1M", label: "Input USD / 1M tokens", type: "number", min: 0, step: "0.000001", required: true },
      { name: "outputUsdPer1M", label: "Output USD / 1M tokens", type: "number", min: 0, step: "0.000001", required: true },
    ],
    secrets: [{ name: "apiKey", label: "API key" }],
  },
  SEEDANCE: {
    title: "Seedance",
    description: "Tạo video qua BytePlus ModelArk và nhận callback tenant-scoped.",
    fields: [
      { name: "baseUrl", label: "Endpoint", type: "url", required: true },
      { name: "model", label: "Model", required: true },
      { name: "apiVersion", label: "API version", required: true },
      { name: "resolution", label: "Resolution", type: "select", options: ["480p", "720p", "1080p", "4k"], required: true },
      { name: "aspectRatio", label: "Aspect ratio", type: "select", options: ["9:16", "16:9", "1:1"], required: true },
      { name: "callbackUrl", label: "Callback URL", type: "url", help: "Để trống nếu chỉ dùng polling." },
      { name: "timeoutSeconds", label: "HTTP timeout (giây)", type: "number", min: 1, required: true },
      { name: "pollIntervalSeconds", label: "Poll interval (giây)", type: "number", min: 1, required: true },
      { name: "taskTimeoutSeconds", label: "Task timeout (giây)", type: "number", min: 30, required: true },
      { name: "usdPerSecond", label: "USD / giây", type: "number", min: 0, step: "0.000001", required: true },
    ],
    secrets: [{ name: "apiKey", label: "ModelArk API key" }, { name: "webhookSecret", label: "Webhook secret" }],
  },
  R2: {
    title: "R2 / S3",
    description: "Object storage riêng cho media của khách hàng này.",
    fields: [
      { name: "accountId", label: "Account ID" },
      { name: "bucket", label: "Bucket", required: true },
      { name: "endpoint", label: "Endpoint nội bộ", type: "url", required: true, help: "API, worker và renderer dùng endpoint này." },
      { name: "browserEndpoint", label: "Endpoint trình duyệt", type: "url", help: "Không bắt buộc. Dùng để tạo presigned URL khi endpoint nội bộ không truy cập được từ trình duyệt." },
      { name: "publicBaseUrl", label: "Public base URL", type: "url", help: "Không bắt buộc; media vẫn mặc định private." },
    ],
    secrets: [{ name: "accessKeyId", label: "Access key ID" }, { name: "secretAccessKey", label: "Secret access key" }],
  },
  META: {
    title: "Meta",
    description: "OAuth, Facebook Pages, Instagram Business và Marketing API.",
    fields: [
      { name: "appId", label: "App ID", required: true },
      { name: "apiVersion", label: "Graph API version", required: true },
      { name: "redirectUrl", label: "OAuth redirect URL", type: "url", required: true },
      { name: "graphBaseUrl", label: "Graph endpoint", type: "url", required: true },
      { name: "dialogBaseUrl", label: "Dialog endpoint", type: "url", required: true },
    ],
    secrets: [{ name: "appSecret", label: "App secret" }],
  },
  RENDERER: {
    title: "Renderer",
    description: "Endpoint renderer nội bộ; xác thực service-to-service vẫn thuộc hạ tầng.",
    fields: [{ name: "baseUrl", label: "Endpoint", type: "url", required: true }],
    secrets: [],
  },
};

const formSchema = z.object({
  enabled: z.boolean(),
  settings: z.record(z.string(), z.string()),
  secrets: z.record(z.string(), z.string()),
  clearSecrets: z.array(z.string()),
});
type FormValues = z.infer<typeof formSchema>;

export default function ProvidersPage() {
  const { clientId } = useStudioScope();
  const queryClient = useQueryClient();
  const profile = useQuery({
    queryKey: ["provider-configuration", clientId],
    enabled: Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/provider-configuration", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải cấu hình provider.");
      return data;
    },
  });
  const mode = useMutation({
    mutationFn: async (nextDemoMode: boolean) => {
      if (!profile.data) throw new Error("Cấu hình chưa sẵn sàng.");
      const { data, error } = await api.PUT("/clients/{clientId}/provider-configuration/mode", { params: { path: { clientId } }, body: { demoMode: nextDemoMode, version: profile.data.version } });
      if (error || !data) throw apiError(error, "Không thể đổi provider mode.");
      return data;
    },
    onSuccess: (data) => queryClient.setQueryData(["provider-configuration", clientId], data),
  });

  return <>
    <PageHeader
      eyebrow="Chỉ Admin · Theo khách hàng"
      title="Cấu hình nhà cung cấp"
      description="Cấu hình được lưu theo khách hàng trong database. Secret được mã hóa ở server và trình duyệt chỉ nhận trạng thái đã cấu hình."
      action={profile.data ? <div className="flex flex-wrap items-center gap-3"><Badge tone={profile.data.demoMode ? "warn" : "good"}>{profile.data.demoMode ? "CHẾ ĐỘ DEMO" : "CHẾ ĐỘ LIVE"}</Badge><Button disabled={mode.isPending} className="min-h-11" onClick={() => {
        const next = !profile.data!.demoMode;
        if (!next && !window.confirm("Chuyển khách hàng này sang LIVE MODE? Các thao tác provider sau đó có thể phát sinh chi phí và vẫn cần approval.")) return;
        mode.mutate(next);
      }}>{mode.isPending ? "Đang cập nhật…" : profile.data.demoMode ? "Chuyển sang live" : "Chuyển sang demo"}</Button></div> : undefined}
    />
    {!clientId ? <StatePanel title="Chọn khách hàng"><p>Chọn khách hàng ở thanh bên để xem và chỉnh đúng bộ provider configuration của khách hàng đó.</p></StatePanel>
      : profile.isLoading ? <SkeletonRows />
        : profile.error ? <StatePanel title="Không thể tải cấu hình" tone="danger"><p role="alert">{profile.error.message}</p><Button className="mt-4" onClick={() => profile.refetch()}>Thử lại</Button></StatePanel>
          : <div className="grid gap-5 xl:grid-cols-2">{profile.data?.providers.map((provider) => <ProviderCard key={`${provider.provider}-${provider.version}`} clientId={clientId} provider={provider} />)}</div>}
    {mode.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{mode.error.message}</p> : null}
    <Card className="mt-6 flex gap-4 p-5"><ShieldCheck className="mt-0.5 size-5 shrink-0 text-[var(--moss)]" /><div><h2 className="font-semibold">Ranh giới bảo mật</h2><p className="mt-1 text-sm leading-6 text-[var(--muted)]">API key và token được gửi qua phiên HTTPS/CSRF hiện tại, mã hóa AES-256-GCM trước khi lưu và không bao giờ được trả lại trình duyệt. Khóa mã hóa gốc và xác thực nội bộ giữa API–renderer vẫn là bí mật hạ tầng nằm ngoài database.</p></div></Card>
  </>;
}

function ProviderCard({ clientId, provider }: { clientId: string; provider: Provider }) {
  const queryClient = useQueryClient();
  const definition = definitions[provider.provider];
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      enabled: provider.enabled,
      settings: Object.fromEntries(Object.entries(provider.settings).map(([key, value]) => [key, String(value ?? "")])),
      secrets: Object.fromEntries(definition.secrets.map((secret) => [secret.name, ""])),
      clearSecrets: [],
    },
  });
  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const settings: Record<string, SettingValue> = {};
      for (const field of definition.fields) {
        const value = (values.settings[field.name] ?? "").trim();
        if (field.type === "number") {
          const number = Number(value);
          if (!Number.isFinite(number)) throw new Error(`${field.label} phải là số hợp lệ.`);
          settings[field.name] = number;
        } else {
          settings[field.name] = value;
        }
      }
      const { data, error } = await api.PUT("/clients/{clientId}/provider-configuration/{provider}", {
        params: { path: { clientId, provider: provider.provider } },
        body: { enabled: values.enabled, settings, secrets: values.secrets, clearSecrets: values.clearSecrets as Array<"apiKey" | "webhookSecret" | "accessKeyId" | "secretAccessKey" | "appSecret">, version: provider.version },
      });
      if (error || !data) throw apiError(error, "Không thể lưu cấu hình provider.");
      return data;
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["provider-configuration", clientId], data);
    },
  });
  const configured = new Set<string>(provider.configuredSecretFields);
  const disabled = !useWatch({ control: form.control, name: "enabled" });
  return <Card className="overflow-hidden">
    <div className="flex items-start justify-between gap-4 border-b border-[var(--line)] p-6">
      <div className="flex gap-4"><span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]">{!provider.enabled ? <CircleOff className="size-5 text-[var(--muted)]" /> : provider.configured ? <CheckCircle2 className="size-5 text-[var(--moss)]" /> : <XCircle className="size-5 text-[var(--coral)]" />}</span><div><h2 className="font-serif text-2xl font-bold">{definition.title}</h2><p className="mt-1 text-sm leading-6 text-[var(--muted)]">{definition.description}</p></div></div>
      <Badge tone={!provider.enabled ? "neutral" : provider.configured ? "good" : "danger"}>{!provider.enabled ? "DISABLED" : provider.configured ? "CONFIGURED" : "INCOMPLETE"}</Badge>
    </div>
    <form className="grid gap-5 p-6" onSubmit={form.handleSubmit((values) => mutation.mutate(values))} noValidate>
      <label className="flex min-h-11 items-center gap-3 rounded-2xl border border-[var(--line)] bg-white px-4 text-sm font-semibold"><input type="checkbox" className="size-5 accent-[var(--moss)]" {...form.register("enabled")} />Cho phép dùng provider này cho khách hàng</label>
      <fieldset disabled={disabled || mutation.isPending} className="grid gap-4 disabled:opacity-55"><legend className="sr-only">{definition.title} settings</legend>
        <div className="grid gap-4 md:grid-cols-2">{definition.fields.map((field) => <SettingField key={field.name} field={field} form={form} suggestionListId={`${provider.provider}-${field.name}-suggestions`} />)}</div>
        {definition.secrets.length ? <div className="grid gap-4 rounded-2xl border border-[var(--line)] bg-[#f7f7f1] p-4"><div className="flex items-center gap-2"><KeyRound className="size-4 text-[var(--moss)]" /><h3 className="text-sm font-bold">Secrets</h3></div>{definition.secrets.map((secret) => {
          const hasValue = configured.has(secret.name);
          return <div key={secret.name} className="grid gap-2"><Field label={secret.label}><input type="password" autoComplete="new-password" className={inputClass} placeholder={hasValue ? "Đã lưu — để trống để giữ nguyên" : "Nhập secret"} {...form.register(`secrets.${secret.name}`)} /></Field>{hasValue ? <label className="flex min-h-11 items-center gap-2 text-xs font-semibold text-[var(--muted)]"><input type="checkbox" value={secret.name} className="size-4 accent-[var(--coral)]" {...form.register("clearSecrets")} />Xóa secret đang lưu</label> : null}</div>;
        })}</div> : null}
      </fieldset>
      {mutation.error ? <p role="alert" className="text-sm font-semibold text-[var(--coral)]">{mutation.error.message}</p> : null}
      {mutation.isSuccess ? <p role="status" className="text-sm font-semibold text-[var(--moss)]">Đã lưu cấu hình {definition.title} cho khách hàng này.</p> : null}
      <div className="flex justify-end"><Button type="submit" disabled={mutation.isPending || !form.formState.isDirty}>{mutation.isPending ? "Đang mã hóa và lưu…" : "Lưu cấu hình"}</Button></div>
    </form>
  </Card>;
}

function SettingField({ field, form, suggestionListId }: { field: FieldDefinition; form: ReturnType<typeof useForm<FormValues>>; suggestionListId: string }) {
  const registration = form.register(`settings.${field.name}`, { required: field.required ? `${field.label} là bắt buộc.` : false });
  const error = form.formState.errors.settings?.[field.name]?.message;
  const suggestions = providerSettingSuggestions[field.name] ?? [];
  return <Field label={field.label} error={typeof error === "string" ? error : undefined}>
    {field.type === "select" ? <select className={inputClass} {...registration}>{field.options?.map((option) => <option key={option} value={option}>{option}</option>)}</select>
      : <><input className={inputClass} type={field.type ?? "text"} min={field.min} step={field.step} list={suggestions.length ? suggestionListId : undefined} {...registration} />{suggestions.length ? <datalist id={suggestionListId}>{suggestions.map((suggestion) => <option key={suggestion} value={suggestion} />)}</datalist> : null}{field.help ? <span className="text-xs font-normal text-[var(--muted)]">{field.help}</span> : null}</>}
  </Field>;
}
