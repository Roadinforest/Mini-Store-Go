import clsx from "clsx";

const steps = ["User Login", "Shipping Address", "Payment Method", "Place Order"];

export function CheckoutSteps({ current = 0 }: { current?: number }) {
  return (
    <div className="mb-10 flex flex-col items-center gap-2 md:flex-row md:justify-between">
      {steps.map((step, index) => (
        <div key={step} className="flex items-center gap-2">
          <div
            className={clsx(
              "w-56 rounded-full p-2 text-center text-sm",
              index === current && "bg-secondary",
            )}
          >
            {step}
          </div>
          {index < steps.length - 1 && <hr className="mx-2 hidden w-16 border-t border-gray-300 md:block" />}
        </div>
      ))}
    </div>
  );
}
