import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import * as api from "@/lib/api";
import type { Order } from "@/lib/types";
import { formatCurrency, formatDate } from "@/lib/utils";

export function OrderDetailsPage() {
  const { id } = useParams();
  const { currentUser, markOrderDelivered, markOrderPaid } = useStore();
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    const orderID = id;

    async function load() {
      const result = await api.getOrderByID(orderID);
      if (cancelled) return;
      setOrder(result.data ?? null);
      setMessage(result.success ? "" : result.message);
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (loading) return <div className="rounded-2xl border p-5 text-sm text-muted-foreground">Loading order...</div>;
  if (!order) return <div>{message || "Order not found."}</div>;

  const currentOrder = order;
  const isAdmin = currentUser?.role === "admin";

  async function onMarkPaid() {
    const result = await markOrderPaid(currentOrder.id);
    setMessage(result.message);
    if (result.success) {
      const refreshed = await api.getOrderByID(currentOrder.id);
      if (refreshed.success && refreshed.data) setOrder(refreshed.data);
    }
  }

  async function onMarkDelivered() {
    const result = await markOrderDelivered(currentOrder.id);
    setMessage(result.message);
    if (result.success) {
      const refreshed = await api.getOrderByID(currentOrder.id);
      if (refreshed.success && refreshed.data) setOrder(refreshed.data);
    }
  }

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <div className="space-y-6 lg:col-span-2">
        <section className="rounded-3xl border p-5">
          <h1 className="h2-bold mb-4">Order Details</h1>
          <div className="text-sm text-muted-foreground">Order ID: {currentOrder.id}</div>
          <div className="mt-3 text-sm text-muted-foreground">Created at {formatDate(currentOrder.createdAt)}</div>
        </section>
        <section className="rounded-3xl border p-5">
          <h2 className="mb-3 text-lg font-semibold">Items</h2>
          <table className="w-full">
            <thead>
              <tr>
                <th className="pb-3 text-left">Item</th>
                <th className="pb-3 text-left">Quantity</th>
                <th className="pb-3 text-right">Price</th>
              </tr>
            </thead>
            <tbody>
              {currentOrder.orderitems.map((item) => (
                <tr key={item.productId} className="border-t">
                  <td className="py-3">
                    <Link to={`/product/${item.slug}`} className="flex items-center gap-3">
                      <img src={item.image} alt={item.name} className="h-12 w-12 object-cover" />
                      <span>{item.name}</span>
                    </Link>
                  </td>
                  <td className="py-3">{item.qty}</td>
                  <td className="py-3 text-right">{formatCurrency(item.price)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      </div>

      <aside className="rounded-3xl border p-5">
        <div className="space-y-3 text-sm">
          <div className="flex-between">
            <span>Payment</span>
            <span>{currentOrder.paymentMethod}</span>
          </div>
          <div className="flex-between">
            <span>Paid</span>
            <span>{currentOrder.isPaid ? formatDate(currentOrder.paidAt) : "No"}</span>
          </div>
          <div className="flex-between">
            <span>Delivered</span>
            <span>{currentOrder.isDelivered ? formatDate(currentOrder.deliveredAt) : "No"}</span>
          </div>
          <div className="flex-between">
            <span>Total</span>
            <span className="font-semibold">{formatCurrency(currentOrder.totalPrice)}</span>
          </div>
        </div>

        {isAdmin && (
          <div className="mt-5 grid gap-3">
            <Button onClick={() => void onMarkPaid()} disabled={currentOrder.isPaid}>
              Mark paid
            </Button>
            <Button variant="outline" onClick={() => void onMarkDelivered()} disabled={!currentOrder.isPaid || currentOrder.isDelivered}>
              Mark delivered
            </Button>
          </div>
        )}
        {message && <div className="mt-3 text-sm text-muted-foreground">{message}</div>}
      </aside>
    </div>
  );
}
