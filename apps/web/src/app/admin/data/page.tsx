import { DataManagementPanel } from "@/features/data-management/data-management-panel";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/admin/data");
}

export default function DataPage() {
  return <DataManagementPanel />;
}
