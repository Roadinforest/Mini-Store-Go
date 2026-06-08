import { DollarSign, Headset, ShoppingBag, WalletCards } from "lucide-react";

const items = [
  {
    icon: ShoppingBag,
    title: "Free Shipping",
    text: "Free shipping on orders above $100",
  },
  {
    icon: DollarSign,
    title: "Money Back Guarantee",
    text: "Within 30 days of purchase",
  },
  {
    icon: WalletCards,
    title: "Flexible Payment",
    text: "Pay with credit card, PayPal or COD",
  },
  {
    icon: Headset,
    title: "24/7 Support",
    text: "Get support at any time",
  },
];

export function IconBoxes() {
  return (
    <section className="rounded-lg border bg-card">
      <div className="grid gap-4 p-4 md:grid-cols-4">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <div key={item.title} className="space-y-2">
              <Icon className="h-5 w-5" />
              <div className="text-sm font-bold">{item.title}</div>
              <div className="text-sm text-muted-foreground">{item.text}</div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
