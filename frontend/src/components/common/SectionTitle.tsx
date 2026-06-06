import type { PropsWithChildren } from "react";

export function SectionTitle({ children }: PropsWithChildren) {
  return <h2 className="h2-bold mb-4">{children}</h2>;
}
