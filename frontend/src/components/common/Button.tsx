import clsx from "clsx";
import type { ButtonHTMLAttributes, PropsWithChildren } from "react";

type Props = PropsWithChildren<
  ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "secondary" | "outline" | "danger";
  }
>;

export function Button({ className, variant = "primary", children, ...props }: Props) {
  return (
    <button
      className={clsx(
        "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50",
        variant === "primary" && "bg-primary text-primary-foreground hover:opacity-90",
        variant === "secondary" && "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        variant === "outline" && "border bg-white hover:bg-muted",
        variant === "danger" && "bg-red-600 text-white hover:bg-red-500",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
