import { useParams } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { formatCurrency, formatDate } from "@/lib/utils";

export function OrderDetailsPage() {
  const { id } = useParams();
  const { state, currentUser, markOrderDelivered, markOrderPaid } = useStore();
  const order = state.orders.find((item) => item.id === id);

  if (!order) return <div>Order not found.</div>;

  const isAdmin = currentUser?.role === "admin";

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <div className="space-y-6 lg:col-span-2">
        <section className="rounded-3xl border p-5">
          <h1 className="h2-bold mb-4">Order {order.id.slice(0, 8)}</h1>
          <div className="text-sm text-muted-foreground">Created at {formatDate(order.createdAt)}</div>
        </section>
        <section className="rounded-3xl border p-5">
          <h2 className="mb-3 text-lg font-semibold">Items</h2>
          <div className="space-y-3">
            {order.orderitems.map((item) => (
              <div key={item.productId} className="flex-between text-sm">
                <span>
                  {item.name} x {item.qty}
                </span>
                <span>{formatCurrency(item.qty * item.price)}</span>
              </div>
            ))}
          </div>
        </section>
      </div>

      <aside className="rounded-3xl border p-5">
        <div className="space-y-3 text-sm">
          <div className="flex-between">
            <span>Payment</span>
            <span>{order.paymentMethod}</span>
          </div>
          <div className="flex-between">
            <span>Paid</span>
            <span>{order.isPaid ? formatDate(order.paidAt) : "No"}</span>
          </div>
          <div className="flex-between">
            <span>Delivered</span>
            <span>{order.isDelivered ? formatDate(order.deliveredAt) : "No"}</span>
          </div>
          <div className="flex-between">
            <span>Total</span>
            <span className="font-semibold">{formatCurrency(order.totalPrice)}</span>
          </div>
        </div>

        {isAdmin && (
          <div className="mt-5 grid gap-3">
            <Button onClick={() => markOrderPaid(order.id)} disabled={order.isPaid}>
              Mark paid
            </Button>
            <Button variant="outline" onClick={() => markOrderDelivered(order.id)} disabled={!order.isPaid || order.isDelivered}>
              Mark delivered
            </Button>
          </div>
        )}
      </aside>
    </div>
  );
}
