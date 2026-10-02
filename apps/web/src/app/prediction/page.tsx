import { getTranslations } from "next-intl/server";
import { PredictionSearchPage } from "@/features/prediction/prediction-search-page";

export async function generateMetadata() {
  const t = await getTranslations("common.metadata.prediction");
  return { title: t("title"), description: t("description") };
}

export default function Page() {
  return <PredictionSearchPage />;
}
