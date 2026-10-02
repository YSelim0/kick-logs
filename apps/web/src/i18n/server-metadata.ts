import { getTranslations } from "next-intl/server";
import { localizeRouteMetadata } from "./route-metadata";

export async function getPageMetadata(pathname: string) {
  return localizeRouteMetadata(pathname, await getTranslations("common.metadata"));
}
