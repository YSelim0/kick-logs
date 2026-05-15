import { UserProfilePage } from "@/features/user-profile/user-profile-page";

type UserProfileRouteProps = {
  params: {
    slug: string;
  };
};

export default function UserProfileRoute({ params }: UserProfileRouteProps) {
  return <UserProfilePage slug={params.slug} />;
}
