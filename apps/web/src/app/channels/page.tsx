import { getPageMetadata } from "@/i18n/server-metadata";
import { ChannelsIndexPage } from "@/features/channel-profile/channels-index-page";

export function generateMetadata() {
  return getPageMetadata("/channels");
}

export default function Page() {
  return <ChannelsIndexPage />;
}
