import { ChannelProfilePage } from "@/features/channel-profile/channel-profile-page";

import { getPageMetadata } from "@/i18n/server-metadata";

type ChannelProfileRouteProps = {
  params: {
    slug: string;
  };
};

export function generateMetadata({ params }: ChannelProfileRouteProps) {
  return getPageMetadata(`/channels/${encodeURIComponent(params.slug)}`);
}

export default function ChannelProfileRoute({ params }: ChannelProfileRouteProps) {
  return <ChannelProfilePage slug={params.slug} />;
}
