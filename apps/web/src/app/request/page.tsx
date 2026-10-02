import { RequestPage } from "@/features/requests/request-page";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/request");
}

export default function PublicRequestPage() {
  return <RequestPage />;
}
