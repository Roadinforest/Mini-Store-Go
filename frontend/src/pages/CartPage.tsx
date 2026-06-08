import { Link, useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { formatCurrency } from "@/lib/utils";
import { useState } from "react";

export function CartPage() {
  const navigate = useNavigate();
  const { state, addToCart, removeFromCart } = useStore();
  const { cart } = state;
  const [message, setMessage] = useState("");

  if (cart.items.length === 0) {
    return (
      <div>
        Cart is empty. <Link to="/">Go shopping</Link>
      </div>
    );
  }

  return (
    <div className="grid gap-5 md:grid-cols-4">
      <div className="overflow-x-auto md:col-span-3">
        <table className="w-full">
          <thead>
            <tr className="text-left text-sm text-muted-foreground">
              <th>Item</th>
              <th className="text-center">Quantity</th>
              <th className="text-right">Price</th>
            </tr>
          </thead>
          <tbody>
            {cart.items.map((item) => (
              <tr key={item.productId} className="border-t">
                <td className="p-3">
                  <Link to={`/product/${item.slug}`} className="flex items-center gap-3">
                    <img src={item.image} alt={item.name} className="h-14 w-14 rounded-md object-cover" />
                    <span>{item.name}</span>
                  </Link>
                </td>
                <td className="p-3">
                  <div className="flex-center gap-3">
                    <Button variant="outline" onClick={async () => setMessage((await removeFromCart(item.productId)).message)}>
                      -
                    </Button>
                    <span>{item.qty}</span>
                    <Button variant="outline" onClick={async () => setMessage((await addToCart(item.productId)).message)}>
                      +
                    </Button>
                  </div>
                </td>
                <td className="p-3 text-right font-semibold">
                  {formatCurrency(item.price)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="rounded-3xl border p-5">
        <div className="pb-3 text-xl">
          Subtotal ({cart.items.reduce((sum, item) => sum + item.qty, 0)}):{" "}
          <span className="font-bold">{formatCurrency(cart.itemsPrice)}</span>
        </div>
        <div className="space-y-2 pb-4 text-sm text-muted-foreground">
          <div className="flex-between">
            <span>Shipping</span>
            <span>{formatCurrency(cart.shippingPrice)}</span>
          </div>
          <div className="flex-between">
            <span>Tax</span>
            <span>{formatCurrency(cart.taxPrice)}</span>
          </div>
          <div className="flex-between font-semibold text-foreground">
            <span>Total</span>
            <span>{formatCurrency(cart.totalPrice)}</span>
          </div>
        </div>
        <Button className="w-full" onClick={() => navigate("/shipping-address")}>
          Proceed to checkout
        </Button>
        {message && <div className="mt-3 text-sm text-muted-foreground">{message}</div>}
      </div>
    </div>
  );
}
