import { FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { PAYMENT_METHODS } from "@/lib/utils";

export function PaymentMethodPage() {
  const navigate = useNavigate();
  const { currentUser, setPaymentMethod } = useStore();

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    setPaymentMethod(String(formData.get("paymentMethod")));
    navigate("/place-order");
  }

  return (
    <div className="mx-auto max-w-xl rounded-3xl border p-6">
      <h1 className="h2-bold mb-4">Payment Method</h1>
      <form onSubmit={onSubmit} className="grid gap-4">
        {PAYMENT_METHODS.map((method) => (
          <label key={method} className="flex items-center gap-3 rounded-xl border p-4">
            <input
              type="radio"
              name="paymentMethod"
              value={method}
              defaultChecked={currentUser?.paymentMethod === method || (!currentUser?.paymentMethod && method === "PayPal")}
            />
            <span>{method}</span>
          </label>
        ))}
        <Button type="submit">Continue</Button>
      </form>
    </div>
  );
}
