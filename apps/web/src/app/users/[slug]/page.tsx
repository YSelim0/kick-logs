import { UserProfilePage } from "@/features/user-profile/user-profile-page";

import { getPageMetadata } from "@/i18n/server-metadata";

type UserProfileRouteProps = {
  params: {
    slug: string;
  };
};

export function generateMetadata({ params }: UserProfileRouteProps) {
  return getPageMetadata(`/users/${encodeURIComponent(params.slug)}`);
}

export default function UserProfileRoute({ params }: UserProfileRouteProps) {
  return <UserProfilePage slug={params.slug} />;
}
