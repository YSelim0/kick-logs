"use client";
import { LoaderCircle } from "lucide-react";
import { useTranslations } from "next-intl";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const BAR_HEIGHTS = [34, 54, 42, 68, 48, 74, 38, 58, 46, 84, 52, 70, 44, 62];

export function ProfileLoading({ kind }: { kind: "channel" | "user" }) {
  const t = useTranslations("common.profile");
  const isChannel = kind === "channel";
  const label = t(kind);

  return (
    <div className="space-y-5">
      <p
        aria-atomic="true"
        aria-live="polite"
        className="flex min-h-5 items-center gap-2 font-mono text-[12px] text-muted-foreground"
        role="status"
      >
        <LoaderCircle
          aria-hidden="true"
          className="h-4 w-4 shrink-0 text-accent motion-safe:animate-spin motion-reduce:animate-none"
        />
        {t(isChannel ? "channelLoading" : "userLoading")}
      </p>

      {/* Keep the live announcement outside the busy region so it is not deferred. */}
      <section aria-busy="true" aria-label={label}>
        <div aria-hidden="true" className="space-y-5" data-testid="profile-placeholders">
          <div className="rounded-lg border border-border bg-panel px-6 py-5">
            <div className="flex flex-wrap items-center justify-between gap-5">
              <div className="flex w-full min-w-0 items-center gap-4 sm:w-auto sm:flex-1">
                <SkeletonBlock
                  className={cn(
                    "h-[72px] w-[72px] shrink-0",
                    isChannel ? "rounded-md" : "rounded-full"
                  )}
                  data-testid="profile-avatar"
                />
                <div className="min-w-0 flex-1 space-y-2">
                  <SkeletonBlock className="h-[22px] w-40 max-w-full" />
                  {!isChannel ? <SkeletonBlock className="h-3 w-24 max-w-full" /> : null}
                  <SkeletonBlock className="h-3 w-64 max-w-full" />
                </div>
              </div>
              <div className="flex w-full flex-wrap gap-2 sm:w-auto">
                <SkeletonBlock className="h-10 w-10 shrink-0 rounded-md" />
                <SkeletonBlock className="h-10 w-full rounded-md sm:w-36" />
              </div>
            </div>
          </div>

          <div
            className={cn(
              "grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border",
              isChannel ? "md:grid-cols-3 xl:grid-cols-6" : "md:grid-cols-4"
            )}
            data-testid="profile-metrics"
          >
            {Array.from({ length: isChannel ? 6 : 4 }, (_, index) => (
              <div className="min-w-0 bg-panel px-5 py-4" key={index}>
                <SkeletonBlock className="h-[15px] w-16 max-w-full" />
                <SkeletonBlock
                  className="mt-2 h-6 w-24 max-w-full"
                  style={{ animationDelay: `${index * -100}ms` }}
                />
              </div>
            ))}
          </div>

          <div className="grid grid-cols-1 gap-5 lg:grid-cols-3" data-testid="profile-analytics">
            {[0, 1, 2].map((panel) => (
              <div className="min-w-0 rounded-lg border border-border bg-panel p-5" key={panel}>
                <SkeletonHeading />
                {panel === 0 ? (
                  <div className="flex h-44 items-end gap-1">
                    {BAR_HEIGHTS.map((height, index) => (
                      <SkeletonBlock
                        className="min-w-0 flex-1 rounded-b-none"
                        key={index}
                        style={{ height: `${height}%`, animationDelay: `${index * -100}ms` }}
                      />
                    ))}
                  </div>
                ) : (
                  <div className="flex h-44 flex-col justify-between">
                    {Array.from({ length: 5 }, (_, index) => (
                      <div className="flex min-w-0 items-center gap-2.5 py-1" key={index}>
                        <SkeletonBlock className="h-3 w-5 shrink-0" />
                        <SkeletonBlock className="h-5 w-5 shrink-0" />
                        <SkeletonBlock
                          className="h-3 min-w-0 flex-1"
                          style={{ animationDelay: `${index * -100}ms` }}
                        />
                        <SkeletonBlock className="h-3 w-10 shrink-0" />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>

          <div className="rounded-lg border border-border bg-panel p-5">
            <SkeletonHeading />
            <div className="divide-y divide-border" data-testid="profile-messages">
              {Array.from({ length: 4 }, (_, index) => (
                <div
                  className="grid grid-cols-1 gap-2 px-3 py-2.5 md:grid-cols-[120px_minmax(0,1fr)_auto] md:gap-4"
                  key={index}
                >
                  <SkeletonBlock className="h-4 w-20 max-w-full" />
                  <SkeletonBlock
                    className={cn("h-4", index % 2 === 0 ? "w-full" : "w-3/4")}
                    style={{ animationDelay: `${index * -100}ms` }}
                  />
                  <SkeletonBlock className="h-3 w-24 max-w-full justify-self-end" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}

function SkeletonHeading() {
  return (
    <div className="mb-4 space-y-1">
      <SkeletonBlock className="h-3.5 w-28 max-w-full" />
      <SkeletonBlock className="h-2.5 w-16 max-w-full" />
    </div>
  );
}

function SkeletonBlock({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      {...props}
      className={cn(
        "block rounded-sm bg-elevated motion-safe:animate-pulse motion-reduce:animate-none",
        className
      )}
    />
  );
}
