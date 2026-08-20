import { describe, expect, it } from "vitest";
import { clientFormSchema } from "./client-form";

const valid = {
  companyName: "An Khang Travel",
  contactName: "Lan Anh",
  contactEmail: "lan@example.com",
  phone: "0900000000",
  industry: "Du lịch",
  market: "Việt Nam",
  internalNotes: "Khách hàng nội bộ",
};

describe("client form", () => {
  it("normalizes surrounding whitespace before submission", () => {
    const result = clientFormSchema.parse({ ...valid, companyName: "  An Khang Travel  " });
    expect(result.companyName).toBe("An Khang Travel");
  });

  it("rejects an invalid company name and contact email", () => {
    const result = clientFormSchema.safeParse({ ...valid, companyName: "A", contactEmail: "not-an-email" });
    expect(result.success).toBe(false);
    if (!result.success) expect(result.error.issues.map((issue) => issue.path[0])).toEqual(expect.arrayContaining(["companyName", "contactEmail"]));
  });

  it("allows an empty optional contact email", () => {
    expect(clientFormSchema.safeParse({ ...valid, contactEmail: "" }).success).toBe(true);
  });
});
