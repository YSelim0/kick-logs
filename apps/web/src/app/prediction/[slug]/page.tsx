import { PredictionAnalysisPage } from "@/features/prediction/prediction-analysis-page";

import { getPageMetadata } from "@/i18n/server-metadata";

type PredictionRouteProps = {
  params: {
    slug: string;
  };
};

export function generateMetadata({ params }: PredictionRouteProps) {
  return getPageMetadata(`/prediction/${encodeURIComponent(params.slug)}`);
}

export default function PredictionRoute({ params }: PredictionRouteProps) {
  return <PredictionAnalysisPage slug={params.slug} />;
}
