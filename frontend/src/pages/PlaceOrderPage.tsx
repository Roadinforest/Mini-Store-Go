import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { formatCurrency } from "@/lib/utils";

export function PlaceOrderPage() {
  const navigate = useNavigate();
  const { state, currentUser, placeOrder } = useStore();
  const [message, setMessage] = useState("");

  async function submitOrder() {
    const result = await placeOrder();
    setMessage(result.message);
    if (result.success && result.orderId) {
      navigate(`/order/${result.orderId}`);
    }
  }

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <div className="space-y-6 lg:col-span-2">
        <section className="rounded-3xl border p-5">
          <h2 className="mb-3 text-lg font-semibold">Shipping</h2>
          <div className="text-sm text-muted-foreground">
            {currentUser?.address
              ? `${currentUser.address.fullName}, ${currentUser.address.streetAddress}, ${currentUser.address.city}, ${currentUser.address.postalCode}, ${currentUser.address.country}`
              : "No shipping address"}
          </div>
        </section>
        <section className="rounded-3xl border p-5">
          <h2 className="mb-3 text-lg font-semibold">Payment</h2>
          <div className="text-sm text-muted-foreground">{currentUser?.paymentMethod ?? "No payment method"}</div>
        </section>
        <section className="rounded-3xl border p-5">
          <h2 className="mb-3 text-lg font-semibold">Items</h2>
          <div className="space-y-3">
            {state.cart.items.map((item) => (
              <div key={item.productId} className="flex-between text-sm">
                <span>
                  {item.name} x {item.qty}
                </span>
                <span>{formatCurrency(item.price * item.qty)}</span>
              </div>
            ))}
          </div>
        </section>
      </div>
      <aside className="rounded-3xl border p-5">
        <h2 className="mb-4 text-lg font-semibold">Order Summary</h2>
        <div className="space-y-3 text-sm">
          <div className="flex-between">
            <span>Items</span>
            <span>{formatCurrency(state.cart.itemsPrice)}</span>
          </div>
          <div className="flex-between">
            <span>Shipping</span>
            <span>{formatCurrency(state.cart.shippingPrice)}</span>
          </div>
          <div className="flex-between">
            <span>Tax</span>
            <span>{formatCurrency(state.cart.taxPrice)}</span>
          </div>
          <div className="flex-between border-t pt-3 font-semibold">
            <span>Total</span>
            <span>{formatCurrency(state.cart.totalPrice)}</span>
          </div>
        </div>
        <Button className="mt-5 w-full" onClick={submitOrder}>
          Place order
        </Button>
        {message && <div className="mt-3 text-sm text-muted-foreground">{message}</div>}
      </aside>
    </div>
  );
}
