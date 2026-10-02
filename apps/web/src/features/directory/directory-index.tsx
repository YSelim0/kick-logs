"use client";

/* eslint-disable @next/next/no-img-element */

import { ChevronDown, Hash, Loader2, MessageSquarePlus, Search, SearchX, User } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { useTranslations } from "next-intl";

import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import {
  useDirectorySearch,
  validDirectoryPrefix
} from "@/features/directory/use-directory-search";
import { buildChannelProfileHref } from "@/lib/channel-profile-slugs";
import { buildUserProfileHref } from "@/lib/kick-profile-slugs";
import type { DirectoryIdentity, DirectoryKind } from "@/types/directory";

export function DirectoryIndex({ kind }: { kind: DirectoryKind }) {
  const t = useTranslations("directory");
  const common = useTranslations("common");
  const [query, setQuery] = useState("");
  const directory = useDirectorySearch(kind);
  const users = kind === "users";
  const Icon = users ? User : Hash;
  const loading = directory.state === "loading";

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute={kind} />
      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <h1 className="mb-5 text-[22px] font-semibold leading-none text-foreground">
          {t(`${kind}.title`)}
        </h1>
        <form
          className="mb-5 flex max-w-lg gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void directory.search(query);
          }}
        >
          <div className="relative min-w-0 flex-1">
            <Search
              aria-hidden
              className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <input
              aria-label={t(`${kind}.search`)}
              autoComplete="off"
              className="h-10 w-full rounded-md border border-border bg-elevated pl-9 pr-3 text-[13px] text-foreground placeholder:text-muted-foreground focus:border-border-strong focus:outline-none"
              id={`${kind}-search-input`}
              minLength={2}
              maxLength={160}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t(`${kind}.placeholder`)}
              spellCheck={false}
              type="search"
              value={query}
            />
          </div>
          <button
            className="flex h-10 w-32 shrink-0 items-center justify-center gap-2 rounded-md bg-accent px-3 text-[13px] font-medium text-text-on-accent transition-colors hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-50"
            disabled={!validDirectoryPrefix(query) || loading}
            type="submit"
          >
            {loading ? (
              <Loader2
                aria-hidden
                className="h-4 w-4 motion-safe:animate-spin motion-reduce:animate-none"
              />
            ) : (
              <Search aria-hidden className="h-4 w-4" />
            )}
            {loading ? t("searching") : common("actions.search")}
          </button>
        </form>

        {directory.state === "idle" ? (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <Icon aria-hidden className="mb-4 h-6 w-6 text-muted-foreground" />
            <p className="text-[15px] font-medium text-foreground">{t(`${kind}.idle`)}</p>
          </div>
        ) : null}
        {loading ? (
          <p
            className="flex items-center gap-2 py-8 font-mono text-[13px] text-muted-foreground"
            role="status"
          >
            <Loader2
              aria-hidden
              className="h-4 w-4 motion-safe:animate-spin motion-reduce:animate-none"
            />{" "}
            {t("searching")}
          </p>
        ) : null}
        {directory.state === "empty" && users ? (
          <p className="py-8 text-[13px] text-muted-foreground" role="status">
            {t("users.empty", { query: directory.submittedQuery })}
          </p>
        ) : null}
        {directory.state === "empty" && !users ? (
          <section
            aria-labelledby="channel-empty-title"
            className="flex max-w-lg gap-3 rounded-lg border border-border bg-panel p-5 sm:p-6"
          >
            <SearchX aria-hidden className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
            <div className="min-w-0 flex-1">
              <h2
                className="text-[14px] font-semibold leading-6 text-foreground [overflow-wrap:anywhere]"
                id="channel-empty-title"
                role="status"
              >
                {t("channels.empty", { query: directory.submittedQuery })}
              </h2>
              <p className="mt-2 text-[13px] leading-6 text-muted-foreground">{t("requestInfo")}</p>
              <Button
                asChild
                className="mt-4 w-full bg-accent text-[13px] text-accent-foreground hover:bg-accent-hover sm:w-auto"
              >
                <Link href="/request">
                  <MessageSquarePlus aria-hidden className="h-4 w-4" />
                  {t("request")}
                </Link>
              </Button>
            </div>
          </section>
        ) : null}
        {directory.items.length > 0 ? (
          <section
            aria-label={t(`${kind}.results`)}
            className="divide-y divide-border border-y border-border"
          >
            {directory.items.map((identity) => (
              <IdentityRow key={identity.id} identity={identity} kind={kind} />
            ))}
          </section>
        ) : null}
        {directory.error ? (
          <p className="py-4 text-[13px] text-danger" role="alert">
            {directory.error === "unavailable"
              ? t("loadError")
              : common(`errors.${directory.error}`)}
          </p>
        ) : null}
        {directory.nextCursor ? (
          <div className="flex justify-center py-4">
            <button
              className="flex h-10 min-w-44 items-center justify-center gap-2 rounded-md border border-border-strong px-4 text-[13px] text-foreground hover:bg-elevated disabled:cursor-not-allowed disabled:opacity-50"
              disabled={directory.loadingMore}
              onClick={() => void directory.loadMore()}
              type="button"
            >
              {directory.loadingMore ? (
                <Loader2
                  aria-hidden
                  className="h-4 w-4 motion-safe:animate-spin motion-reduce:animate-none"
                />
              ) : (
                <ChevronDown aria-hidden className="h-4 w-4" />
              )}
              {directory.loadingMore ? common("actions.loading") : t("loadMore")}
            </button>
          </div>
        ) : null}
      </div>
    </main>
  );
}

function IdentityRow({ identity, kind }: { identity: DirectoryIdentity; kind: DirectoryKind }) {
  const [imageFailed, setImageFailed] = useState(false);
  const users = kind === "users";
  const href = users ? buildUserProfileHref(identity.slug) : buildChannelProfileHref(identity.slug);
  const shape = users ? "rounded-full" : "rounded-md";
  const content = (
    <>
      {identity.profile_image_url && !imageFailed ? (
        <img
          alt={identity.name}
          className={`h-10 w-10 shrink-0 border border-border object-cover ${shape}`}
          height={40}
          onError={() => setImageFailed(true)}
          src={identity.profile_image_url}
          width={40}
        />
      ) : (
        <div
          aria-hidden
          className={`flex h-10 w-10 shrink-0 items-center justify-center border border-border bg-elevated font-mono text-sm font-semibold text-muted-foreground ${shape}`}
        >
          {identity.name.charAt(0).toUpperCase() || (users ? "U" : "#")}
        </div>
      )}
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">
          {users ? "@" : ""}
          {identity.name}
        </p>
        {identity.slug && (!users || identity.slug !== identity.name) ? (
          <p className="truncate font-mono text-[11px] text-muted-foreground">
            {users ? "" : "#"}
            {identity.slug}
          </p>
        ) : null}
      </div>
    </>
  );
  const className =
    "flex items-center gap-3 px-3 py-3 transition-colors hover:bg-elevated sm:gap-4 sm:px-4";
  return href ? (
    <Link className={className} href={href}>
      {content}
    </Link>
  ) : (
    <div className={className}>{content}</div>
  );
}
