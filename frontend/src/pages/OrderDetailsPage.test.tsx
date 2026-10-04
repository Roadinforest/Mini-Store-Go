import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import * as api from "@/lib/api";
import type { Order } from "@/lib/types";
import { OrderDetailsPage } from "./OrderDetailsPage";

const { markOrderPaid } = vi.hoisted(() => ({ markOrderPaid: vi.fn() }));
vi.mock("@/lib/api", () => ({ getOrderByID: vi.fn() }));
vi.mock("@/app/store", () => ({
  useStore: () => ({ currentUser: { role: "admin" }, markOrderPaid, markOrderDelivered: vi.fn() }),
}));

afterEach(() => { cleanup(); vi.useRealTimers(); vi.resetAllMocks(); });

it("closes the payment action on an already-open page when the deadline arrives", async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-10-04T12:00:00Z"));
  const order: Order = {
    id: "order-1", userId: "user-1", shippingAddress: { fullName: "Test", streetAddress: "Street", city: "City", postalCode: "123", country: "Test" },
    paymentMethod: "cash", itemsPrice: 10, shippingPrice: 0, taxPrice: 0, totalPrice: 10,
    isPaid: false, paidAt: null, isDelivered: false, deliveredAt: null, status: "UNPAID",
    expiresAt: "2026-10-04T12:00:01Z", expiredAt: null, createdAt: "2026-10-04T11:45:01Z", orderitems: [],
  };
  vi.mocked(api.getOrderByID).mockResolvedValue({ success: true, message: "OK", data: order });
  await act(async () => {
    render(<MemoryRouter initialEntries={["/order/order-1"]}><Routes><Route path="/order/:id" element={<OrderDetailsPage />} /></Routes></MemoryRouter>);
  });
  const button = screen.getByRole<HTMLButtonElement>("button", { name: "Mark paid" });
  expect(button.disabled).toBe(false);
  await act(async () => { vi.advanceTimersByTime(1000); });
  expect(button.disabled).toBe(true);
  expect(screen.getByText("Status: Expired")).toBeTruthy();
  expect(screen.getByText("The payment window has closed. Please place a new order.")).toBeTruthy();
  expect(markOrderPaid).not.toHaveBeenCalled();
});
