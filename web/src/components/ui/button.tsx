import * as React from "react";
import { cn } from "@/lib/utils";

type ButtonVariant = "primary" | "secondary" | "tertiary";

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
};

const variants: Record<ButtonVariant, string> = {
  primary:
    "primary-gradient text-white hover:opacity-95 active:scale-[0.98] ambient-shadow",
  secondary:
    "bg-transparent text-[var(--primary)] ghost-border hover:bg-[var(--surface-low)]",
  tertiary:
    "bg-[var(--surface-high)] text-[var(--foreground)] hover:bg-[var(--surface-variant)]",
};

export function Button({
  className,
  variant = "primary",
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        "inline-flex items-center justify-center rounded-md px-5 py-2.5 text-sm font-semibold transition-all",
        variants[variant],
        className,
      )}
      {...props}
    />
  );
}
