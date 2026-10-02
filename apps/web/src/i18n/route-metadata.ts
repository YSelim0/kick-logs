import type { useTranslations } from "next-intl";

type MetadataTranslator = ReturnType<typeof useTranslations<"common.metadata">>;

const pages = {
  "/": "home",
  "/search": "search",
  "/channels": "channels",
  "/users": "users",
  "/prediction": "prediction",
  "/request": "request",
  "/login": "login",
  "/admin": "adminOperations",
  "/admin/operations": "adminOperations",
  "/admin/channels": "adminChannels",
  "/admin/users": "adminUsers",
  "/admin/requests": "adminRequests",
  "/admin/data": "adminData"
} as const;

const profiles = {
  channels: "channelProfile",
  users: "userProfile",
  prediction: "predictionAnalysis"
} as const;

export function localizeRouteMetadata(pathname: string, t: MetadataTranslator) {
  const path = pathname.replace(/\/$/, "") || "/";
  const profile = /^\/(channels|users|prediction)\/([^/]+)$/.exec(path);
  let title: string;
  if (profile) {
    let slug = profile[2];
    try {
      slug = decodeURIComponent(slug);
    } catch {
      // A malformed URL must not break the document metadata.
    }
    title = t(`titles.${profiles[profile[1] as keyof typeof profiles]}`, { slug });
  } else {
    title = t(`titles.${pages[path as keyof typeof pages] ?? "notFound"}`);
  }

  const description =
    path === "/channels"
      ? t("channels.description")
      : path === "/users"
        ? t("users.description")
        : path === "/prediction"
          ? t("prediction.description")
          : t("description");

  return { title: `${title} - KickLogs`, description };
}
