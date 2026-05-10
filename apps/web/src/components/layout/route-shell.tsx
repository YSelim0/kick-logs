import Link from "next/link";
import type { ReactNode } from "react";

type RouteShellProps = {
  title: string;
  eyebrow: string;
  children?: ReactNode;
};

const routes = [
  { href: "/", label: "Root" },
  { href: "/search", label: "Search" },
  { href: "/login", label: "Login" },
  { href: "/admin", label: "Admin" }
];

export function RouteShell({ title, eyebrow, children }: RouteShellProps) {
  return (
    <main className="min-h-screen bg-background px-6 py-6 text-foreground">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <header className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-4">
          <Link href="/search" className="text-sm font-semibold text-primary">
            Kick Logs
          </Link>
          <nav className="flex flex-wrap gap-2 text-xs text-muted-foreground">
            {routes.map((route) => (
              <Link
                className="rounded-md border border-border px-3 py-2 hover:border-primary hover:text-primary"
                href={route.href}
                key={route.href}
              >
                {route.label}
              </Link>
            ))}
          </nav>
        </header>

        <section className="rounded-lg border border-border bg-black p-6">
          <p className="mb-2 text-xs font-medium uppercase text-accent">{eyebrow}</p>
          <h1 className="text-2xl font-semibold tracking-normal">{title}</h1>
          {children ? <div className="mt-4 text-sm text-muted-foreground">{children}</div> : null}
        </section>
      </div>
    </main>
  );
}
