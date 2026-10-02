import { getTranslations } from "next-intl/server";
import { ChannelsIndexPage } from "@/features/channel-profile/channels-index-page";

export async function generateMetadata() {
  const t = await getTranslations("common.metadata.channels");
  return { title: t("title"), description: t("description") };
}

export default function Page() {
  return <ChannelsIndexPage />;
}
