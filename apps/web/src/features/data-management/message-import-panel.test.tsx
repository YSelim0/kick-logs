import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MessageImportPanel } from "@/features/data-management/message-import-panel";
import type { MessageImportPreview, MessageImportResult } from "@/types/api";

const importMocks = vi.hoisted(() => ({
  confirmMessageImport: vi.fn(),
  previewMessageImport: vi.fn()
}));

vi.mock("@/features/data-management/api", () => importMocks);

describe("MessageImportPanel", () => {
  beforeEach(() => {
    importMocks.confirmMessageImport.mockReset();
    importMocks.previewMessageImport.mockReset();
    importMocks.previewMessageImport.mockResolvedValue(previewFixture());
    importMocks.confirmMessageImport.mockResolvedValue(resultFixture());
  });

  it("previews the selected export file with the given limit", async () => {
    render(<MessageImportPanel />);

    const file = exportFile();
    fireEvent.change(screen.getByLabelText("EXPORT DOSYASI (JSON)"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));

    await waitFor(() => expect(importMocks.previewMessageImport).toHaveBeenCalledWith(file, 20));

    expect(await screen.findByText("İçe Aktarma Önizleme Sonucu")).toBeInTheDocument();
    expect(screen.getByText("EKLENECEK")).toBeInTheDocument();
    expect(screen.getByText("ZATEN MEVCUT")).toBeInTheDocument();
    expect(screen.getByText("DOSYA İÇİ TEKRAR")).toBeInTheDocument();
    expect(screen.getByText("HATALI")).toBeInTheDocument();
  });

  it("sends an empty limit as null so the whole file is analyzed", async () => {
    render(<MessageImportPanel />);

    const file = exportFile();
    fireEvent.change(screen.getByLabelText("EXPORT DOSYASI (JSON)"), { target: { files: [file] } });
    fireEvent.change(screen.getByLabelText("LİMİT (BOŞ = TÜMÜ)"), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));

    await waitFor(() => expect(importMocks.previewMessageImport).toHaveBeenCalledWith(file, null));
  });

  it("keeps the import button disabled until the confirmation text matches", async () => {
    render(<MessageImportPanel />);

    const file = exportFile();
    fireEvent.change(screen.getByLabelText("EXPORT DOSYASI (JSON)"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));
    await screen.findByText("İçe Aktarma Önizleme Sonucu");

    const importButton = screen.getByRole("button", { name: "İçe aktar" });
    expect(importButton).toBeDisabled();

    fireEvent.change(screen.getByLabelText(/ONAY METNİ/), { target: { value: "yanlis" } });
    expect(importButton).toBeDisabled();

    fireEvent.change(screen.getByLabelText(/ONAY METNİ/), { target: { value: "IMPORT MESSAGES" } });
    expect(importButton).toBeEnabled();

    fireEvent.click(importButton);
    await waitFor(() =>
      expect(importMocks.confirmMessageImport).toHaveBeenCalledWith(file, 20, "IMPORT MESSAGES")
    );
    expect(await screen.findByText(/İçe aktarma tamamlandı/)).toBeInTheDocument();
  });

  it("hides the confirmation form when the preview cannot execute", async () => {
    importMocks.previewMessageImport.mockResolvedValue({
      ...previewFixture(),
      to_insert: 0,
      can_execute: false,
      reason: "No new messages to import."
    });
    render(<MessageImportPanel />);

    fireEvent.change(screen.getByLabelText("EXPORT DOSYASI (JSON)"), {
      target: { files: [exportFile()] }
    });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));

    await screen.findByText("İçe Aktarma Önizleme Sonucu");
    expect(screen.getByText("No new messages to import.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "İçe aktar" })).not.toBeInTheDocument();
  });

  it("asks for a file before previewing", async () => {
    render(<MessageImportPanel />);

    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));

    expect(await screen.findByText("Önce bir JSON export dosyası seçin.")).toBeInTheDocument();
    expect(importMocks.previewMessageImport).not.toHaveBeenCalled();
  });

  it("shows an error when the preview request fails", async () => {
    importMocks.previewMessageImport.mockRejectedValue(new Error("boom"));
    render(<MessageImportPanel />);

    fireEvent.change(screen.getByLabelText("EXPORT DOSYASI (JSON)"), {
      target: { files: [exportFile()] }
    });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));

    expect(await screen.findByText("Önizleme başarısız oldu.")).toBeInTheDocument();
  });
});

function exportFile() {
  return new File(['{"items":[]}'], "export.json", { type: "application/json" });
}

function previewFixture(): MessageImportPreview {
  return {
    total_in_file: 429,
    records_read: 20,
    limit: 20,
    to_insert: 18,
    already_exists: 1,
    duplicate_in_file: 1,
    invalid: 0,
    invalid_reasons: [],
    sample_to_insert_ids: ["msg-1", "msg-2"],
    confirmation_text: "IMPORT MESSAGES",
    can_execute: true,
    reason: null
  };
}

function resultFixture(): MessageImportResult {
  return {
    written: 18,
    already_exists: 1,
    duplicate_in_file: 1,
    invalid: 0,
    confirmation_text: "IMPORT MESSAGES"
  };
}
