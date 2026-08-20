import { z } from "zod";

export const clientFormSchema = z.object({
  companyName: z.string().trim().min(2, "Tên công ty cần ít nhất 2 ký tự.").max(200),
  contactName: z.string().trim().max(160),
  contactEmail: z.union([z.email("Email không hợp lệ."), z.literal("")]),
  phone: z.string().trim().max(40),
  industry: z.string().trim().max(160),
  market: z.string().trim().max(160),
  internalNotes: z.string().trim().max(10000),
});

export type ClientFormValues = z.infer<typeof clientFormSchema>;
