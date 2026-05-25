import { PredictionSearchPage } from "@/features/prediction/prediction-search-page";

export const metadata = {
  title: "Prediction — Kick Logs",
  description: "Bir Kick kanalının son tahmin oyununu görüntüleyin."
};

export default function Page() {
  return <PredictionSearchPage />;
}
