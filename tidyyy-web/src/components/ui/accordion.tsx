import * as React from "react";
import { cn } from "@/lib/utils";

type AccordionItemProps = {
  title: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
};

export function Accordion({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("space-y-4", className)} {...props} />;
}

export function AccordionItem({ title, children, defaultOpen }: AccordionItemProps) {
  return (
    <details
      className="group overflow-hidden rounded-xl bg-(--surface-lowest) ghost-border"
      open={defaultOpen}
    >
      <summary className="flex cursor-pointer list-none items-center justify-between p-6 text-left text-sm font-bold tracking-tight">
        {title}
        <span className="text-lg leading-none transition-transform group-open:rotate-180">+</span>
      </summary>
      <div className="px-6 pb-6 text-sm leading-relaxed text-[color-mix(in_srgb,var(--foreground)_72%,white)]">
        {children}
      </div>
    </details>
  );
}
