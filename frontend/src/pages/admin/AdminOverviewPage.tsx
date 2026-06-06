import { useStore } from "@/app/store";
import { formatCurrency } from "@/lib/utils";

export function AdminOverviewPage() {
  const { state } = useStore();
  const totalSales = state.orders.filter((order) => order.isPaid).reduce((sum, order) => sum + order.totalPrice, 0);

  const cards = [
    { label: "Orders", value: String(state.orders.length) },
    { label: "Products", value: String(state.products.length) },
    { label: "Users", value: String(state.users.length) },
    { label: "Sales", value: formatCurrency(totalSales) },
  ];

  return (
    <div className="space-y-6">
      <h1 className="h2-bold">Admin Overview</h1>
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
