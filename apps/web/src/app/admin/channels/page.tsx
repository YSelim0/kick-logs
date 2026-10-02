import { ChannelAdmin } from "@/features/channels/channel-admin";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/admin/channels");
}

export default function ChannelsPage() {
  return <ChannelAdmin />;
}
