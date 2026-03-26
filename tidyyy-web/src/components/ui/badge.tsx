import * as React from "react";
import { cn } from "@/lib/utils";

type BadgeVariant = "default" | "success";

type BadgeProps = React.HTMLAttributes<HTMLSpanElement> & {
  variant?: BadgeVariant;
};

const styles: Record<BadgeVariant, string> = {
  default: "bg-[var(--surface-high)] text-[var(--foreground)]",
  success: "bg-[#c7e7fa] text-[#385565]",
};

export function Badge({
  className,
  variant = "default",
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.05rem]",
        styles[variant],
        className,
      )}
      {...props}
    />
  );
}
