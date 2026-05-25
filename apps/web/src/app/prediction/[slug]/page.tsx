import { PredictionAnalysisPage } from "@/features/prediction/prediction-analysis-page";

type PredictionRouteProps = {
  params: {
    slug: string;
  };
};

export default function PredictionRoute({ params }: PredictionRouteProps) {
  return <PredictionAnalysisPage slug={params.slug} />;
}
