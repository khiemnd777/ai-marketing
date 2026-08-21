import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MediaUploader } from "./media-uploader";

const scope = {
  clientId: "22222222-2222-4222-8222-222222222222",
  workspaceId: "33333333-3333-4333-8333-333333333333",
};

describe("MediaUploader", () => {
  beforeEach(() => {
    Object.defineProperty(URL, "createObjectURL", { configurable: true, writable: true, value: vi.fn(() => "blob:local-preview") });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, writable: true, value: vi.fn() });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("opens the file picker from the whole dropzone and previews the selected image", async () => {
    const onUploaded = vi.fn();
    const { container } = render(<MediaUploader scope={scope} onUploaded={onUploaded} />);
    const input = container.querySelector<HTMLInputElement>('input[type="file"]');
    const dropzone = screen.getByRole("button", { name: /Kéo thả file vào đây/i });
    expect(input).not.toBeNull();

    const click = vi.spyOn(input!, "click");
    fireEvent.click(dropzone);
    expect(click).toHaveBeenCalledOnce();

    const image = new File(["preview"], "vali-hero.png", { type: "image/png" });
    fireEvent.change(input!, { target: { files: [image] } });

    expect(await screen.findByAltText("Xem trước vali-hero.png")).toHaveAttribute("src", "blob:local-preview");
    expect(screen.getByText("1/20 file · kiểm tra nội dung trước khi tải lên")).toBeInTheDocument();
    expect(screen.getByText("HERO_IMAGE")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tải lên 1 file" })).toBeEnabled();
  });

  it("accepts a dragged file and shows a usable queue preview", async () => {
    render(<MediaUploader scope={scope} logoOnly brandId="44444444-4444-4444-8444-444444444444" onUploaded={vi.fn()} />);
    const dropzone = screen.getByRole("button", { name: /Kéo thả file vào đây/i });
    const logo = new File(["logo"], "logo-dang-cong.webp", { type: "image/webp" });

    fireEvent.dragEnter(dropzone, { dataTransfer: { files: [logo] } });
    expect(screen.getByText("Thả file để thêm vào hàng đợi")).toBeInTheDocument();
    fireEvent.drop(dropzone, { dataTransfer: { files: [logo] } });

    expect(await screen.findByAltText("Xem trước logo-dang-cong.webp")).toBeInTheDocument();
    expect(screen.getByText("BRAND_LOGO")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Xóa logo-dang-cong.webp khỏi hàng đợi" })).toBeEnabled();
  });
});
