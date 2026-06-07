import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import * as api from "@/lib/api";
import type { Order } from "@/lib/types";
import { formatCurrency, formatDate } from "@/lib/utils";

export function UserOrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const result = await api.getMyOrders();
      if (cancelled) return;
      setOrders(result.data?.items ?? []);
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="h2-bold">My Orders</h1>
      <div className="overflow-x-auto rounded-3xl border">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left">
            <tr>
              <th className="p-4">Order</th>
              <th className="p-4">Created</th>
              <th className="p-4">Total</th>
              <th className="p-4">Paid</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="p-4 text-muted-foreground" colSpan={4}>
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
                  <td className="p-4">{order.isPaid ? "Yes" : "No"}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
