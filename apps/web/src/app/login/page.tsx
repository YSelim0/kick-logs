import { LoginScreen } from "@/features/auth/login-screen";
import { getPageMetadata } from "@/i18n/server-metadata";

export function generateMetadata() {
  return getPageMetadata("/login");
}

export default function LoginPage() {
  return <LoginScreen />;
}
