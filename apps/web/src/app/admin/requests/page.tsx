import { RequestAdmin } from "@/features/requests/request-admin";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/admin/requests");
}

export default function AdminRequestsPage() {
  return <RequestAdmin />;
}
