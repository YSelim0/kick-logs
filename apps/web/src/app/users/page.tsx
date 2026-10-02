import { getTranslations } from "next-intl/server";
import { UsersIndexPage } from "@/features/user-profile/users-index-page";

export async function generateMetadata() {
  const t = await getTranslations("common.metadata.users");
  return { title: t("title"), description: t("description") };
}

export default function Page() {
  return <UsersIndexPage />;
}
