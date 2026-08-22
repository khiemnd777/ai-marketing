"use client";

import { FileDown } from "lucide-react";
import { Button } from "@/components/ui";

const PDF_DOCUMENT_TITLE = "Bang-gia-dich-vu-AI-Studio-2026-08-22";

export function ExportPdfButton() {
  const handleExport = () => {
    const previousTitle = document.title;
    document.title = PDF_DOCUMENT_TITLE;
    try {
      window.print();
    } finally {
      document.title = previousTitle;
    }
  };

  return (
    <Button
      data-pricing-export-control
      className="min-h-11 gap-2 px-4"
      aria-label="Xuất bảng giá thành PDF"
      onClick={handleExport}
    >
      <FileDown className="size-4" aria-hidden="true" />
      Xuất PDF
    </Button>
  );
}
