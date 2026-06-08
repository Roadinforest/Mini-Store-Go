import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { CheckoutSteps } from "@/components/common/CheckoutSteps";
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
    <>
      <CheckoutSteps current={3} />
      <h1 className="py-4 text-2xl">Place Order</h1>
      <div className="grid gap-6 md:grid-cols-3">
        <div className="space-y-4 overflow-x-auto md:col-span-2">
          <section className="rounded-3xl border p-5">
            <h2 className="pb-4 text-xl">Shipping Address</h2>
            {currentUser?.address ? (
              <>
                <p>{currentUser.address.fullName}</p>
                <p>
                  {currentUser.address.streetAddress}, {currentUser.address.city} {currentUser.address.postalCode}, {currentUser.address.country}
                </p>
                <div className="mt-3">
                  <Button variant="outline" asChild to="/shipping-address">Edit</Button>
                </div>
              </>
            ) : (
              <p>No shipping address</p>
            )}
          </section>

          <section className="rounded-3xl border p-5">
            <h2 className="pb-4 text-xl">Payment Method</h2>
            <p>{currentUser?.paymentMethod ?? "No payment method"}</p>
            <div className="mt-3">
              <Button variant="outline" asChild to="/payment-method">Edit</Button>
            </div>
          </section>

          <section className="rounded-3xl border p-5">
            <h2 className="pb-4 text-xl">Order Items</h2>
            <table className="w-full">
              <thead>
                <tr>
                  <th className="pb-3 text-left">Item</th>
                  <th className="pb-3 text-left">Quantity</th>
                  <th className="pb-3 text-right">Price</th>
                </tr>
              </thead>
              <tbody>
                {state.cart.items.map((item) => (
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

        <div>
          <section className="rounded-3xl border p-5 space-y-4">
            <div className="flex justify-between">
              <div>Items</div>
              <div>{formatCurrency(state.cart.itemsPrice)}</div>
            </div>
            <div className="flex justify-between">
              <div>Tax</div>
              <div>{formatCurrency(state.cart.taxPrice)}</div>
            </div>
            <div className="flex justify-between">
              <div>Shipping</div>
              <div>{formatCurrency(state.cart.shippingPrice)}</div>
            </div>
            <div className="flex justify-between">
              <div>Total</div>
              <div>{formatCurrency(state.cart.totalPrice)}</div>
            </div>
            <Button className="w-full" onClick={submitOrder}>
              Place order
            </Button>
            {message && <div className="text-sm text-muted-foreground">{message}</div>}
          </section>
        </div>
      </div>
    </>
  );
}
