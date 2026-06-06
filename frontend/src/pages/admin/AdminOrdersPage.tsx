import { Link } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { formatCurrency, formatDate } from "@/lib/utils";

export function AdminOrdersPage() {
  const { state, markOrderDelivered, markOrderPaid } = useStore();

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
            {state.orders.map((order) => (
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
                    <Button variant="outline" onClick={() => markOrderPaid(order.id)} disabled={order.isPaid}>
                      Mark paid
                    </Button>
                    <Button variant="outline" onClick={() => markOrderDelivered(order.id)} disabled={!order.isPaid || order.isDelivered}>
                      Deliver
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
