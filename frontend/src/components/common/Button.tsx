import clsx from "clsx";
import type { ButtonHTMLAttributes, PropsWithChildren, ReactNode } from "react";
import { Link } from "react-router-dom";

type Props = PropsWithChildren<
  ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "secondary" | "outline" | "danger";
    asChild?: boolean;
    to?: string;
  }
>;

function getClassName(variant: Props["variant"], className?: string) {
  return clsx(
    "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50",
    variant === "primary" && "bg-primary text-primary-foreground hover:opacity-90",
    variant === "secondary" && "bg-secondary text-secondary-foreground hover:bg-secondary/80",
    variant === "outline" && "border bg-background hover:bg-muted",
    variant === "danger" && "bg-red-600 text-white hover:bg-red-500",
    className,
  );
}

export function Button({ className, variant = "primary", children, asChild, to, ...props }: Props) {
  if (asChild && to) {
    return (
      <Link to={to} className={getClassName(variant, className)}>
        {children as ReactNode}
      </Link>
    );
  }

  return <button className={getClassName(variant, className)} {...props}>{children}</button>;
}
