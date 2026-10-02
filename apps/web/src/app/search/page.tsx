import { SearchScreen } from "@/features/search/search-screen";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/search");
}

export default function SearchPage() {
  return <SearchScreen />;
}
