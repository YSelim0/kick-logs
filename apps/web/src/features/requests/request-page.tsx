"use client";

import { CheckCircle2, Hash, Mail, MessageSquareText, Send, Type } from "lucide-react";
import { useMemo, useState } from "react";

import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import { createUserRequest } from "@/features/requests/api";
import { ApiClientError } from "@/lib/api-client";
import { cn } from "@/lib/utils";
import type { UserRequestType } from "@/types/api";

type FormState = {
  type: UserRequestType;
  channelSlug: string;
  title: string;
  message: string;
  contact: string;
  website: string;
};

const INITIAL_FORM: FormState = {
  type: "channel_request",
  channelSlug: "",
  title: "",
  message: "",
  contact: "",
  website: ""
};

const MODES: { value: UserRequestType; label: string; description: string }[] = [
  {
    value: "channel_request",
    label: "Kanal Talebi",
    description: "Takip edilmesini istediğin Kick kanalını gönder."
  },
  {
    value: "feedback",
    label: "Geri Bildirim",
    description: "Uygulama fikri, hata bildirimi veya genel mesaj gönder."
  }
];

export function RequestPage() {
  const [form, setForm] = useState<FormState>(INITIAL_FORM);
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [error, setError] = useState<string | null>(null);
  const [requestID, setRequestID] = useState<string | null>(null);

  const isChannelRequest = form.type === "channel_request";
  const canSubmit = useMemo(() => {
    const title = form.title.trim();
    const message = form.message.trim();
    const channelSlug = form.channelSlug.trim();
    return (
      status !== "submitting" &&
      title.length >= 3 &&
      message.length >= 5 &&
      (!isChannelRequest || channelSlug.length >= 2)
    );
  }, [form.channelSlug, form.message, form.title, isChannelRequest, status]);

  async function submitRequest() {
    setStatus("submitting");
    setError(null);
    setRequestID(null);

    try {
      const response = await createUserRequest({
        type: form.type,
        title: form.title.trim(),
        message: form.message.trim(),
        channel_slug: isChannelRequest ? form.channelSlug.trim() : undefined,
        channel_display_name: isChannelRequest ? form.channelSlug.trim() : undefined,
        contact: optionalValue(form.contact),
        website: form.website
      });
      setRequestID(response.request_id);
      setStatus("success");
      setForm({ ...INITIAL_FORM, type: form.type });
    } catch (caught) {
      setError(resolveRequestError(caught));
      setStatus("error");
    }
  }

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="request" />

      <div className="mx-auto flex max-w-[1280px] flex-col gap-6 px-6 py-8 md:py-10">
        <header className="flex flex-col gap-1">
          <h1 className="text-[24px] font-semibold tracking-tight text-foreground">Talep</h1>
          <p className="max-w-2xl text-[13px] leading-relaxed text-muted-foreground">
            Takip edilmesini istediğin kanalı veya uygulama hakkında iletmek istediğin mesajı
            gönder.
          </p>
        </header>

        <section className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
          <form
            className="rounded-lg border border-border bg-panel p-5"
            onSubmit={(event) => {
              event.preventDefault();
              if (canSubmit) {
                void submitRequest();
              }
            }}
          >
            <fieldset className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <legend className="sr-only">Talep tipi</legend>
              {MODES.map((mode) => (
                <button
                  key={mode.value}
                  className={cn(
                    "flex min-h-[86px] flex-col items-start justify-center gap-1 rounded-md border px-4 py-3 text-left transition-colors",
                    form.type === mode.value
                      ? "border-accent bg-accent/10 text-foreground"
                      : "border-border bg-elevated text-muted-foreground hover:border-border-strong hover:text-foreground"
                  )}
                  onClick={() => {
                    setForm((current) => ({
                      ...current,
                      type: mode.value,
                      channelSlug: mode.value === "feedback" ? "" : current.channelSlug
                    }));
                    setStatus("idle");
                    setError(null);
                    setRequestID(null);
                  }}
                  type="button"
                >
                  <span className="flex items-center gap-2 text-[13px] font-semibold">
                    <span
                      aria-hidden
                      className={cn(
                        "h-2 w-2 rounded-full",
                        form.type === mode.value ? "bg-accent" : "bg-faint"
                      )}
                    />
                    {mode.label}
                  </span>
                  <span className="text-[12px] leading-relaxed text-muted-foreground">
                    {mode.description}
                  </span>
                </button>
              ))}
            </fieldset>

            <div className="mt-5 grid grid-cols-1 gap-4">
              {isChannelRequest ? (
                <Field label="Kanal adı" htmlFor="channel-slug" icon={<Hash className="h-4 w-4" />}>
                  <input
                    autoComplete="off"
                    className={inputClassName}
                    id="channel-slug"
                    maxLength={120}
                    onChange={(event) =>
                      setForm((current) => ({ ...current, channelSlug: event.target.value }))
                    }
                    placeholder="Kanal adı veya Kick kanal URL'i"
                    value={form.channelSlug}
                  />
                </Field>
              ) : null}

              <Field label="Başlık" htmlFor="request-title" icon={<Type className="h-4 w-4" />}>
                <input
                  className={inputClassName}
                  id="request-title"
                  maxLength={120}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, title: event.target.value }))
                  }
                  placeholder={
                    isChannelRequest ? "Kanal takip listesine eklensin" : "Kısa bir konu yaz"
                  }
                  value={form.title}
                />
              </Field>

              <Field
                label="Mesaj"
                htmlFor="request-message"
                icon={<MessageSquareText className="h-4 w-4" />}
              >
                <textarea
                  className={cn(inputClassName, "min-h-[150px] resize-y py-3 leading-relaxed")}
                  id="request-message"
                  maxLength={2000}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, message: event.target.value }))
                  }
                  placeholder={
                    isChannelRequest
                      ? "Bu kanal neden takip edilmeli?"
                      : "Görüşünü, hata bildirimini veya istediğin geliştirmeyi yaz."
                  }
                  value={form.message}
                />
              </Field>

              <Field
                label="İletişim"
                htmlFor="request-contact"
                hint="Opsiyonel"
                icon={<Mail className="h-4 w-4" />}
              >
                <input
                  className={inputClassName}
                  id="request-contact"
                  maxLength={120}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, contact: event.target.value }))
                  }
                  placeholder="E-posta veya Discord"
                  value={form.contact}
                />
              </Field>

              <label className="hidden" htmlFor="request-website">
                Website
                <input
                  autoComplete="off"
                  id="request-website"
                  onChange={(event) =>
                    setForm((current) => ({ ...current, website: event.target.value }))
                  }
                  tabIndex={-1}
                  value={form.website}
                />
              </label>
            </div>

            {status === "success" ? (
              <div className="mt-5 rounded-md border border-accent bg-accent/10 px-4 py-3 text-[13px] text-foreground">
                <div className="flex items-start gap-2">
                  <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-accent" />
                  <div className="flex flex-col gap-1">
                    <span className="font-semibold">Talebin alındı.</span>
                    {requestID ? (
                      <span className="font-mono text-2xs uppercase text-muted-foreground">
                        ID {requestID}
                      </span>
                    ) : null}
                  </div>
                </div>
              </div>
            ) : null}

            {status === "error" && error ? (
              <div className="mt-5 rounded-md border border-danger bg-elevated px-4 py-3 text-[13px] text-foreground">
                {error}
              </div>
            ) : null}

            <div className="mt-5 flex flex-col gap-3 border-t border-border pt-5 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-[12px] leading-relaxed text-muted-foreground">
                Gönderimler spam koruması ile korunur.
              </p>
              <Button className="w-full sm:w-auto" disabled={!canSubmit} type="submit">
                <Send className="h-4 w-4" />
                {status === "submitting" ? "Gönderiliyor..." : "Gönder"}
              </Button>
            </div>
          </form>

          <aside className="h-fit rounded-lg border border-border bg-panel p-5">
            <div className="flex flex-col gap-4">
              <div>
                <p className="font-mono text-2xs uppercase text-muted-foreground">Süreç</p>
                <h2 className="mt-1 text-[15px] font-semibold text-foreground">
                  Talebin nasıl değerlendirilir?
                </h2>
              </div>
              <div className="grid gap-3 text-[13px] leading-relaxed text-muted-foreground">
                <p>
                  Kanal talepleri uygunluk, tekrar durumu ve izlenebilirlik açısından kontrol
                  edilir.
                </p>
                <p>
                  Geri bildirimler hata düzeltmeleri, arama deneyimi ve yeni özellik planlaması için
                  dikkate alınır.
                </p>
                <p>
                  İletişim bilgisi bırakırsan yalnızca talebin hakkında ek bilgi gerektiğinde geri
                  dönüş yapılır.
                </p>
                <p>Gönderim yapmak, talebin kesin olarak kabul edildiği anlamına gelmez.</p>
              </div>
            </div>
          </aside>
        </section>
      </div>
    </main>
  );
}

function Field({
  children,
  hint,
  htmlFor,
  icon,
  label
}: {
  children: React.ReactNode;
  hint?: string;
  htmlFor: string;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label
        className="flex items-center justify-between gap-2 font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
        htmlFor={htmlFor}
      >
        <span className="flex items-center gap-1.5">
          <span className="text-accent">{icon}</span>
          {label}
        </span>
        {hint ? <span className="text-faint">{hint}</span> : null}
      </label>
      {children}
    </div>
  );
}

const inputClassName =
  "w-full rounded-md border border-border bg-elevated px-3 py-2.5 text-[13px] text-foreground outline-none transition-colors placeholder:text-faint focus:border-border-strong";

function optionalValue(value: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function resolveRequestError(error: unknown) {
  if (error instanceof ApiClientError) {
    if (error.status === 429) {
      return "Çok fazla talep gönderdin. Kısa süre sonra tekrar dene.";
    }
    if (error.status === 400) {
      return "Form alanlarını kontrol edip tekrar gönder.";
    }
  }

  return "Talep gönderilemedi. Biraz sonra tekrar dene.";
}
