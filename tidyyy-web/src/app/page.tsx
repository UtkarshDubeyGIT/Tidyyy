import { Accordion, AccordionItem } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";

export default function Home() {
  return (
    <div className="bg-(--surface) text-foreground">
      <nav className="sticky top-0 z-50 border-b border-black/5 bg-[color-mix(in_srgb,var(--surface)_80%,transparent)] backdrop-blur-xl">
        <div className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-6 md:px-8">
          <div className="text-xl font-black tracking-tight">Tidyyy</div>
          <div className="hidden items-center gap-8 text-sm font-semibold md:flex">
            <a href="#how">How It Works</a>
            <a href="#privacy">Privacy</a>
            <a href="#faq">FAQ</a>
          </div>
          <Button className="px-4 py-2">Get Started</Button>
        </div>
      </nav>

      <main className="space-y-24 pb-24 pt-16 md:pt-24">
        <section className="mx-auto grid w-full max-w-7xl items-center gap-12 px-6 md:grid-cols-2 md:px-8">
          <div className="space-y-8 fade-up">
            <Badge>V1.0.4 now available</Badge>
            <h1 className="text-5xl font-black leading-[0.92] tracking-tight md:text-7xl">
              Your file system,
              <br />
              finally human.
            </h1>
            <p className="max-w-xl text-lg leading-relaxed text-[color-mix(in_srgb,var(--foreground)_74%,white)]">
              Tidyyy uses local-first AI to automatically rename your downloads,
              screenshots, and documents into clear, searchable filenames.
            </p>
            <div className="flex flex-wrap gap-3">
              <Button className="px-7 py-4">Download for macOS</Button>
              <Button variant="secondary" className="px-7 py-4">
                View Demo
              </Button>
            </div>
          </div>

          <div className="relative fade-up" style={{ animationDelay: "120ms" }}>
            <div className="hero-orb absolute inset-0 rounded-4xl blur-3xl" />
            <Card className="glass-panel ambient-shadow relative rotate-1 rounded-3xl border-white/40 p-7 md:rotate-2">
              <p className="mb-3 text-xs font-bold uppercase tracking-[0.14rem] text-[color-mix(in_srgb,var(--foreground)_62%,white)]">
                Original
              </p>
              <p className="rounded-md bg-(--surface-lowest) p-3 font-mono text-sm ghost-border">
                Screenshot 2026-03-23 at 14.42.11.png
              </p>
              <div className="my-4 flex justify-center text-2xl text-(--primary)">↓</div>
              <p className="mb-3 text-xs font-bold uppercase tracking-[0.14rem] text-(--primary)">
                Renamed
              </p>
              <p className="rounded-md bg-[#c7e7fa] p-3 font-mono text-sm text-[#385565]">
                q1-revenue-dashboard.png
              </p>
            </Card>
          </div>
        </section>

        <section id="how" className="mx-auto w-full max-w-7xl px-6 md:px-8">
          <div className="mb-10 max-w-2xl space-y-3">
            <h2 className="text-4xl font-black tracking-tight">The Precision Pipeline</h2>
            <p className="text-[color-mix(in_srgb,var(--foreground)_74%,white)]">
              Four autonomous stages, built for silent background operation.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-4">
            <Card className="bg-(--surface-low)">
              <CardTitle>01 Watcher</CardTitle>
              <CardDescription>
                Monitors selected folders for new files after settle delay.
              </CardDescription>
            </Card>
            <Card className="bg-(--surface-low)">
              <CardTitle>02 Triage</CardTitle>
              <CardDescription>
                Validates MIME and extension before work is queued.
              </CardDescription>
            </Card>
            <Card className="bg-(--surface-low)">
              <CardTitle>03 Extractor</CardTitle>
              <CardDescription>
                Reads PDF text or OCRs image content for semantic clues.
              </CardDescription>
            </Card>
            <Card className="bg-(--surface-low)">
              <CardTitle>04 Namer</CardTitle>
              <CardDescription>
                Applies naming rules and resolves conflicts with safe suffixes.
              </CardDescription>
            </Card>
          </div>
        </section>

        <section id="privacy" className="mx-auto w-full max-w-7xl px-6 md:px-8">
          <div className="glass-panel ambient-shadow grid gap-8 rounded-[1.8rem] p-8 md:grid-cols-2 md:p-12">
            <div className="space-y-6">
              <h2 className="text-4xl font-black tracking-tight">Local-First AI</h2>
              <p className="text-lg leading-relaxed text-[color-mix(in_srgb,var(--foreground)_74%,white)]">
                Your files stay on your machine by default. Tidyyy runs local
                inference first, supports optional cloud fallback only when
                explicitly enabled, and keeps naming deterministic.
              </p>
              <div className="flex flex-wrap gap-3">
                <Badge variant="success">End-to-end privacy</Badge>
                <Badge>Offline capable</Badge>
              </div>
            </div>
            <div className="flex items-center justify-center">
              <div className="primary-gradient ambient-shadow grid h-52 w-52 place-items-center rounded-full text-6xl text-white">
                Shield
              </div>
            </div>
          </div>
        </section>

        <section id="faq" className="mx-auto w-full max-w-4xl px-6 md:px-8">
          <h2 className="mb-8 text-center text-4xl font-black tracking-tight">Frequently Asked</h2>
          <Accordion>
            <AccordionItem title="How long are generated names?" defaultOpen>
              Tidyyy targets practical 2-5 word slugs to balance readability and
              brevity.
            </AccordionItem>
            <AccordionItem title="What if two files are similar?">
              Collision handling appends safe suffixes like -2 and -3 while
              preserving semantic roots.
            </AccordionItem>
            <AccordionItem title="Will it slow down my machine?">
              Work runs in a lightweight background pipeline with throttling and
              concurrency limits for stability.
            </AccordionItem>
          </Accordion>
        </section>

        <section className="mx-auto w-full max-w-7xl px-6 md:px-8">
          <div className="primary-gradient ambient-shadow space-y-6 rounded-4xl p-10 text-center text-white md:p-16">
            <h2 className="text-4xl font-black tracking-tight md:text-5xl">
              Ready to tidy your digital life?
            </h2>
            <p className="mx-auto max-w-xl text-lg text-white/85">
              Stop losing files to bad names. Install Tidyyy and make your
              workspace searchable again.
            </p>
            <div className="flex flex-col justify-center gap-3 pt-2 md:flex-row">
              <Button
                variant="tertiary"
                className="bg-white px-8 py-4 text-(--primary)"
              >
                Download for Desktop
              </Button>
              <Button
                variant="secondary"
                className="border-white/40 px-8 py-4 text-white hover:bg-white/10"
              >
                See How It Works
              </Button>
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}
