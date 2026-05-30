import { fetchKickPrediction } from "@/features/prediction/kick-prediction-client";
import type { Prediction } from "@/types/api";

export function getPrediction(slug: string): Promise<Prediction> {
  return fetchKickPrediction(slug);
}
