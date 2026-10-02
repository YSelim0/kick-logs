import { redirect } from "next/navigation";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/admin");
}

export default function AdminPage() {
  redirect("/admin/operations");
}
