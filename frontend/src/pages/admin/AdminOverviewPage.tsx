import { useEffect, useState } from "react";
import * as api from "@/lib/api";
import { formatCurrency } from "@/lib/utils";
import type { AdminOverview } from "@/lib/types";

export function AdminOverviewPage() {
  const [overview, setOverview] = useState<AdminOverview | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const result = await api.getAdminOverview();
      if (cancelled) return;
      setOverview(result.data ?? null);
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const cards = [
    { label: "Orders", value: String(overview?.orderCount ?? 0) },
    { label: "Products", value: String(overview?.productCount ?? 0) },
    { label: "Users", value: String(overview?.userCount ?? 0) },
    { label: "Sales", value: formatCurrency(overview?.totalSales ?? 0) },
  ];

  return (
    <div className="space-y-6">
      <h1 className="h2-bold">Admin Overview</h1>
      {loading && <div className="text-sm text-muted-foreground">Loading overview...</div>}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <div key={card.label} className="rounded-3xl border p-5">
            <div className="text-sm uppercase tracking-[0.2em] text-muted-foreground">{card.label}</div>
            <div className="mt-2 text-3xl font-semibold">{card.value}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
