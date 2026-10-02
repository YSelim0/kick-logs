import { LandingPage } from "@/features/landing/landing-page";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/");
}

export default function HomePage() {
  return <LandingPage />;
}
