import { getPageMetadata } from "@/i18n/server-metadata";
import { PredictionSearchPage } from "@/features/prediction/prediction-search-page";

export function generateMetadata() {
  return getPageMetadata("/prediction");
}

export default function Page() {
  return <PredictionSearchPage />;
}
