import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { CheckoutSteps } from "@/components/common/CheckoutSteps";

export function ShippingAddressPage() {
  const navigate = useNavigate();
  const { currentUser, setShippingAddress } = useStore();
  const [message, setMessage] = useState("");

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const result = await setShippingAddress({
      fullName: String(formData.get("fullName")),
      streetAddress: String(formData.get("streetAddress")),
      city: String(formData.get("city")),
      postalCode: String(formData.get("postalCode")),
      country: String(formData.get("country")),
    });
    setMessage(result.message);
    if (result.success) {
      navigate("/payment-method");
    }
  }

  return (
    <>
      <CheckoutSteps current={1} />
      <div className="mx-auto max-w-2xl rounded-3xl border p-6">
        <h1 className="h2-bold mb-4">Shipping Address</h1>
        <form onSubmit={onSubmit} className="grid gap-4 md:grid-cols-2">
          <input name="fullName" defaultValue={currentUser?.address?.fullName ?? ""} placeholder="Full name" className="rounded-md border px-3 py-2" required />
          <input name="streetAddress" defaultValue={currentUser?.address?.streetAddress ?? ""} placeholder="Street address" className="rounded-md border px-3 py-2" required />
          <input name="city" defaultValue={currentUser?.address?.city ?? ""} placeholder="City" className="rounded-md border px-3 py-2" required />
          <input name="postalCode" defaultValue={currentUser?.address?.postalCode ?? ""} placeholder="Postal code" className="rounded-md border px-3 py-2" required />
          <input name="country" defaultValue={currentUser?.address?.country ?? ""} placeholder="Country" className="rounded-md border px-3 py-2 md:col-span-2" required />
          <div className="md:col-span-2">
            <Button type="submit">Continue</Button>
          </div>
        </form>
        {message && <div className="mt-4 text-sm text-muted-foreground">{message}</div>}
      </div>
    </>
  );
}
