import { BarChart3, Check, CircleDollarSign, Clock3, Languages, Megaphone, ShieldCheck, Video } from "lucide-react";
import type { ReactNode } from "react";
import { Badge, Card, PageHeader } from "@/components/ui";
import { formatVnd, pricingTerms, serviceAddOns, servicePackages, type ServicePackage } from "@/lib/service-packages";
import { ExportPdfButton } from "./export-pdf-button";

const languageLabel = { vi: "Tiếng Việt", en: "Tiếng Anh" } as const;
const advertisingLabel = { NOT_INCLUDED: "Không bao gồm", PAUSED_SETUP: "Hỗ trợ thiết lập" } as const;

export default function PricingPage() {
  return (
    <div data-pricing-page>
      <div data-pricing-header>
        <PageHeader
          eyebrow="Gói dịch vụ"
          title="Bảng giá dịch vụ quản lý"
          description="Ba gói được thiết kế theo nhu cầu sản xuất nội dung và video hàng tháng. Mọi chi phí bổ sung đều được thống nhất trước khi thực hiện."
          action={(
            <div className="flex flex-wrap items-center gap-3">
              <Badge tone="warn">Áp dụng từ 22/08/2026</Badge>
              <ExportPdfButton />
            </div>
          )}
        />
      </div>

      <section data-pricing-packages aria-labelledby="pricing-packages-heading">
        <h2 id="pricing-packages-heading" className="sr-only">Các gói dịch vụ</h2>
        <div data-pricing-package-grid className="grid items-stretch gap-5 xl:grid-cols-3">
          {servicePackages.map((item) => <PackageCard key={item.id} item={item} />)}
        </div>
      </section>

      <section data-pricing-comparison className="mt-8" aria-labelledby="comparison-heading">
        <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-xs font-bold uppercase tracking-[0.18em] text-[var(--moss)]">So sánh nhanh</p>
            <h2 id="comparison-heading" className="mt-1 font-serif text-2xl font-bold">Số lượng dịch vụ mỗi tháng</h2>
          </div>
          <p className="max-w-xl text-sm leading-6 text-[var(--muted)]">Mỗi bộ chiến dịch gồm định hướng, ý tưởng, 14 nội dung, kịch bản và hướng dẫn cảnh quay. Số video được tính riêng theo từng gói.</p>
        </div>
        <Card className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[760px] text-left text-sm">
              <caption className="sr-only">So sánh ba gói dịch vụ</caption>
              <thead className="bg-[#edf0e7]">
                <tr>
                  <th scope="col" className="px-5 py-4 font-bold">Hạng mục</th>
                  {servicePackages.map((item) => <th key={item.id} scope="col" className="px-5 py-4 font-serif text-lg">{item.name}</th>)}
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--line)]">
                <ComparisonRow label="Nhóm thương hiệu" values={servicePackages.map((item) => item.workspaces.toString())} />
                <ComparisonRow label="Sản phẩm" values={servicePackages.map((item) => item.activeProducts.toString())} />
                <ComparisonRow label="Phí khởi tạo (một lần)" values={servicePackages.map((item) => <OnboardingPrice key={item.id} item={item} />)} />
                <ComparisonRow label="Bộ chiến dịch" values={servicePackages.map((item) => <PromotionalValue key={item.id} value={item.campaignsPerMonth.toString()} previousValue={item.previousCampaignsPerMonth?.toString()} />)} />
                <ComparisonRow label="Nội dung" values={servicePackages.map((item) => item.contentVariantsPerMonth.toString())} />
                <ComparisonRow label="Video dọc 9:16" values={servicePackages.map((item) => <PromotionalValue key={item.id} value={item.finalVideosPerMonth.toString()} previousValue={item.previousFinalVideosPerMonth?.toString()} />)} />
                <ComparisonRow label="Thời lượng video" values={servicePackages.map((item) => item.videoDurations.map((value) => `${value} giây`).join(" hoặc "))} />
                <ComparisonRow label="Ngôn ngữ" values={servicePackages.map((item) => item.languages.map((value) => languageLabel[value]).join(" / "))} />
                <ComparisonRow label="Quảng cáo Facebook & Instagram" values={servicePackages.map((item) => advertisingLabel[item.metaAds])} />
                <ComparisonRow label="Hỗ trợ định kỳ" values={servicePackages.map((item) => item.reviewCadence)} />
                <ComparisonRow label="Thời gian phản hồi" values={servicePackages.map((item) => item.responseTime)} />
              </tbody>
            </table>
          </div>
        </Card>
      </section>

      <section data-pricing-notes className="mt-8 grid gap-6 xl:grid-cols-[1.1fr_.9fr]" aria-labelledby="commercial-notes-heading">
        <Card className="p-6 md:p-7">
          <div className="flex items-center gap-3">
            <span className="grid size-11 place-items-center rounded-2xl bg-[var(--lime)]"><CircleDollarSign className="size-5" /></span>
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.16em] text-[var(--moss)]">Mua bổ sung</p>
              <h2 id="commercial-notes-heading" className="font-serif text-2xl font-bold">Đơn giá ngoài gói</h2>
            </div>
          </div>
          <div className="mt-5 grid gap-3">
            {serviceAddOns.map((item) => (
              <div key={item.name} className="grid gap-2 rounded-2xl border border-[var(--line)] bg-white/65 p-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start">
                <div><h3 className="font-bold">{item.name}</h3><p className="mt-1 text-xs leading-5 text-[var(--muted)]">{item.note}</p></div>
                <strong className="text-sm text-[var(--moss)] sm:text-right">{item.price}</strong>
              </div>
            ))}
          </div>
        </Card>

        <Card className="border-[#c9d5b9] bg-[#f6f8ed] p-6 md:p-7">
          <div className="flex items-center gap-3">
            <span className="grid size-11 place-items-center rounded-2xl bg-white"><ShieldCheck className="size-5 text-[var(--moss)]" /></span>
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.16em] text-[var(--moss)]">Điều kiện áp dụng</p>
              <h2 className="font-serif text-2xl font-bold">Rõ chi phí trước khi chạy</h2>
            </div>
          </div>
          <ul className="mt-5 grid gap-3 text-sm leading-6 text-[var(--muted)]">
            {pricingTerms.map((term) => <li key={term} className="flex gap-3"><Check className="mt-1 size-4 shrink-0 text-[var(--moss)]" aria-hidden="true" /><span>{term}</span></li>)}
          </ul>
          <p className="mt-5 rounded-2xl bg-white/80 p-4 text-sm font-semibold leading-6">Khuyến nghị hợp đồng tối thiểu 3 tháng để có đủ chu kỳ sản xuất, phân phối và dữ liệu tối ưu; không áp dụng gia hạn tự động.</p>
        </Card>
      </section>
    </div>
  );
}

function PackageCard({ item }: { item: ServicePackage }) {
  return (
    <Card data-pricing-card className={item.recommended ? "relative flex h-full flex-col border-[var(--moss)] bg-[#fbfff1] p-6 ring-2 ring-[var(--moss)]/10 md:p-7" : "flex h-full flex-col p-6 md:p-7"}>
      <div className="flex min-h-6 flex-wrap gap-2">
        {item.recommended ? <Badge tone="good">Khuyến nghị</Badge> : null}
        {item.promotionLabel ? <Badge tone="warn">{item.promotionLabel}</Badge> : null}
      </div>
      <h3 className="mt-4 font-serif text-3xl font-bold">{item.name}</h3>
      <p className="mt-2 min-h-12 text-sm leading-6 text-[var(--muted)]">{item.audience}</p>
      <div data-pricing-price className="mt-5 border-y border-[var(--line)] py-5">
        <p className="font-serif text-4xl font-bold tracking-tight">{formatVnd(item.monthlyFeeVnd)}</p>
        <p className="mt-1 text-sm font-semibold text-[var(--muted)]">mỗi tháng</p>
        <p data-pricing-onboarding className="mt-2 flex flex-wrap items-baseline gap-x-2 gap-y-1 text-sm font-semibold text-[var(--muted)]">
          <span>Phí khởi tạo (một lần):</span><OnboardingPrice item={item} />
        </p>
      </div>
      <dl className="mt-5 grid grid-cols-2 gap-3">
        <PackageMetric icon={Megaphone} label="Chiến dịch" value={`${item.campaignsPerMonth}/tháng`} previousValue={item.previousCampaignsPerMonth ? `${item.previousCampaignsPerMonth}/tháng` : undefined} />
        <PackageMetric icon={Video} label="Video" value={`${item.finalVideosPerMonth}/tháng`} previousValue={item.previousFinalVideosPerMonth ? `${item.previousFinalVideosPerMonth}/tháng` : undefined} />
        <PackageMetric icon={Languages} label="Ngôn ngữ" value={item.languages.map((value) => languageLabel[value]).join(" / ")} />
        <PackageMetric icon={Clock3} label="Thời lượng" value={item.videoDurations.map((value) => `${value} giây`).join(" / ")} />
        <PackageMetric className="col-span-2" icon={Megaphone} label="Quảng cáo Facebook & Instagram" value={advertisingLabel[item.metaAds]} />
      </dl>
      <ul data-pricing-features className="mt-6 grid gap-3 text-sm leading-6">
        {item.features.map((feature) => <li key={feature} className="flex gap-3"><Check className="mt-1 size-4 shrink-0 text-[var(--moss)]" aria-hidden="true" /><span>{feature}</span></li>)}
      </ul>
      <div data-pricing-review className="mt-6 flex items-center gap-2 rounded-2xl bg-[#edf0e7] px-4 py-3 text-xs font-semibold text-[var(--muted)]"><BarChart3 className="size-4 shrink-0" />{item.reviewCadence}</div>
    </Card>
  );
}

function PackageMetric({ icon: Icon, label, value, previousValue, className = "" }: { icon: typeof Video; label: string; value: string; previousValue?: string; className?: string }) {
  return <div className={`rounded-2xl bg-[#f2f4ed] p-3 ${className}`}><dt className="flex items-center gap-2 text-xs font-bold text-[var(--muted)]"><Icon className="size-3.5" />{label}</dt><dd className="mt-2 font-bold"><PromotionalValue value={value} previousValue={previousValue} /></dd></div>;
}

function OnboardingPrice({ item }: { item: ServicePackage }) {
  const value = item.onboardingFeeVnd === 0 ? "Miễn phí" : formatVnd(item.onboardingFeeVnd);
  const previousValue = item.previousOnboardingFeeVnd ? formatVnd(item.previousOnboardingFeeVnd) : undefined;
  return <span className="inline-flex flex-wrap items-baseline gap-2"><PromotionalValue value={value} previousValue={previousValue} />{item.onboardingDiscountLabel ? <Badge tone="warn">{item.onboardingDiscountLabel}</Badge> : null}</span>;
}

function PromotionalValue({ value, previousValue }: { value: string; previousValue?: string }) {
  const accessibleLabel = previousValue ? `${value}, ưu đãi từ ${previousValue}` : undefined;
  return <span className="inline-flex flex-wrap items-baseline gap-2" aria-label={accessibleLabel}>{previousValue ? <s aria-hidden="true" className="font-medium text-[var(--muted)] decoration-[var(--coral)] decoration-2">{previousValue}</s> : null}<span aria-hidden={previousValue ? "true" : undefined} className={previousValue ? "font-black text-[var(--moss)]" : undefined}>{value}</span></span>;
}

function ComparisonRow({ label, values }: { label: string; values: readonly ReactNode[] }) {
  return <tr><th scope="row" className="px-5 py-4 font-semibold">{label}</th>{values.map((value, index) => <td key={`${label}-${servicePackages[index]!.id}`} className="px-5 py-4 text-[var(--muted)]">{value}</td>)}</tr>;
}
