import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/common/Button";
import * as api from "@/lib/api";
import type { Order } from "@/lib/types";
import { formatCurrency, formatDate } from "@/lib/utils";

export function AdminOrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const result = await api.getAdminOrders();
      if (cancelled) return;
      setOrders(result.data?.items ?? []);
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  async function onMarkPaid(orderId: string) {
    const result = await api.markOrderPaid(orderId);
    setMessage(result.message);
    if (result.success && result.data) {
      setOrders((current) => current.map((order) => (order.id === orderId ? result.data! : order)));
    }
  }

  async function onMarkDelivered(orderId: string) {
    const result = await api.markOrderDelivered(orderId);
    setMessage(result.message);
    if (result.success && result.data) {
      setOrders((current) => current.map((order) => (order.id === orderId ? result.data! : order)));
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="h2-bold">Orders</h1>
      <div className="overflow-x-auto rounded-3xl border">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left">
            <tr>
              <th className="p-4">Order</th>
              <th className="p-4">Created</th>
              <th className="p-4">Total</th>
              <th className="p-4">Status</th>
              <th className="p-4">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="p-4 text-muted-foreground" colSpan={5}>
                  Loading orders...
                </td>
              </tr>
            ) : (
              orders.map((order) => (
                <tr key={order.id} className="border-t">
                  <td className="p-4">
                    <Link to={`/order/${order.id}`}>{order.id.slice(0, 8)}</Link>
                  </td>
                  <td className="p-4">{formatDate(order.createdAt)}</td>
                  <td className="p-4">{formatCurrency(order.totalPrice)}</td>
                  <td className="p-4">
                    {order.isPaid ? "Paid" : "Pending"} / {order.isDelivered ? "Delivered" : "Shipping"}
                  </td>
                  <td className="p-4">
                    <div className="flex gap-2">
                      <Button variant="outline" onClick={() => void onMarkPaid(order.id)} disabled={order.isPaid}>
                        Mark paid
                      </Button>
                      <Button variant="outline" onClick={() => void onMarkDelivered(order.id)} disabled={!order.isPaid || order.isDelivered}>
                        Deliver
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {message && <div className="text-sm text-muted-foreground">{message}</div>}
    </div>
  );
}
