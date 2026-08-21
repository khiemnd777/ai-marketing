import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { OptionPicker } from "./option-picker";

const options = [
  { value: "fact-a", label: "Trọng lượng · 2.9 kg" },
  { value: "fact-b", label: "Dung tích · 38 L" },
];

describe("OptionPicker", () => {
  it("shows labels instead of ids and returns the selected ids", () => {
    const onChange = vi.fn();
    render(<OptionPicker label="Product facts bắt buộc" options={options} value={[]} onChange={onChange} />);

    expect(screen.getByText("Trọng lượng · 2.9 kg")).toBeInTheDocument();
    expect(screen.queryByText("fact-a")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Trọng lượng · 2.9 kg" }));
    expect(onChange).toHaveBeenCalledWith(["fact-a"]);
  });

  it("supports a single-choice radio mode", () => {
    const onChange = vi.fn();
    render(<OptionPicker label="Chọn một" options={options} value={["fact-a"]} onChange={onChange} multiple={false} />);

    fireEvent.click(screen.getByRole("radio", { name: "Dung tích · 38 L" }));
    expect(onChange).toHaveBeenCalledWith(["fact-b"]);
  });
});
