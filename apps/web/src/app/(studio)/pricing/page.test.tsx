import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import PricingPage from "./page";

afterEach(cleanup);

describe("PricingPage", () => {
  it("presents three transparent managed-service packages without a self-service checkout", () => {
    render(<PricingPage />);

    expect(screen.getByRole("heading", { level: 1, name: "Bảng giá dịch vụ quản lý" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3, name: "Khởi động" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3, name: "Tăng trưởng" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3, name: "Mở rộng" })).toBeInTheDocument();
    expect(screen.getByText("Khuyến nghị")).toBeInTheDocument();
    expect(screen.getByText("5.290.000đ")).toBeInTheDocument();
    expect(screen.getByText("12.900.000đ")).toBeInTheDocument();
    expect(screen.getByText("24.900.000đ")).toBeInTheDocument();
    expect(screen.getByText("8 video dọc hoàn chỉnh 30 giây đã qua duyệt")).toBeInTheDocument();
    expect(screen.getAllByText("Không bao gồm báo cáo định kỳ")).toHaveLength(2);
    expect(screen.getAllByText("Hỗ trợ thiết lập")).toHaveLength(4);
    expect(document.querySelectorAll("s")).toHaveLength(8);
    expect(screen.getByRole("table", { name: "So sánh ba gói dịch vụ" })).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(/PAUSED|Meta Ads|quota|provider|render|entitlement/i);
    expect(screen.queryByRole("button", { name: /đăng ký|thanh toán|mua gói/i })).not.toBeInTheDocument();
  });
});
