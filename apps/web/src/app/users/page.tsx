import { getPageMetadata } from "@/i18n/server-metadata";
import { UsersIndexPage } from "@/features/user-profile/users-index-page";

export function generateMetadata() {
  return getPageMetadata("/users");
}

export default function Page() {
  return <UsersIndexPage />;
}
