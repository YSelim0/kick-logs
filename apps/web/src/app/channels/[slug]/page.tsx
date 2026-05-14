import { ChannelProfilePage } from "@/features/channel-profile/channel-profile-page";

type ChannelProfileRouteProps = {
  params: {
    slug: string;
  };
};

export default function ChannelProfileRoute({ params }: ChannelProfileRouteProps) {
  return <ChannelProfilePage slug={params.slug} />;
}
