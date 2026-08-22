import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExportPdfButton } from "./export-pdf-button";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("ExportPdfButton", () => {
  it("opens the browser PDF dialog and restores the page title", () => {
    const previousTitle = document.title;
    document.title = "AI Product Marketing Studio";
    const print = vi.spyOn(window, "print").mockImplementation(() => undefined);

    render(<ExportPdfButton />);
    fireEvent.click(screen.getByRole("button", { name: "Xuất bảng giá thành PDF" }));

    expect(print).toHaveBeenCalledOnce();
    expect(document.title).toBe("AI Product Marketing Studio");
    document.title = previousTitle;
  });
});
