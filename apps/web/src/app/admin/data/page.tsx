import { DataManagementPanel } from "@/features/data-management/data-management-panel";
import { MessageImportPanel } from "@/features/data-management/message-import-panel";

export default function DataPage() {
  return (
    <div className="flex flex-col gap-6">
      <DataManagementPanel />
      <MessageImportPanel />
    </div>
  );
}
