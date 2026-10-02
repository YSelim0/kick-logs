import { UserAdmin } from "@/features/users/user-admin";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/admin/users");
}

export default function UsersPage() {
  return <UserAdmin />;
}
