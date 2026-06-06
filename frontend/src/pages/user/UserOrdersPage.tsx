import { Link } from "react-router-dom";
import { useStore } from "@/app/store";
import { formatCurrency, formatDate } from "@/lib/utils";

export function UserOrdersPage() {
  const { currentUser, state } = useStore();
  const orders = state.orders.filter((order) => order.userId === currentUser?.id);

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
            {orders.map((order) => (
              <tr key={order.id} className="border-t">
                <td className="p-4">
                  <Link to={`/order/${order.id}`}>{order.id.slice(0, 8)}</Link>
                </td>
                <td className="p-4">{formatDate(order.createdAt)}</td>
                <td className="p-4">{formatCurrency(order.totalPrice)}</td>
                <td className="p-4">{order.isPaid ? "Yes" : "No"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
