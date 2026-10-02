import { NotFoundScreen } from "@/components/not-found-screen";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/_not-found");
}

export default function NotFound() {
  return <NotFoundScreen />;
}
