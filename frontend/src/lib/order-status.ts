import type { Order } from "@/lib/types";

export function isOrderExpired(order: Order, now: number): boolean {
  return !order.isPaid && (order.status === "EXPIRED" || order.expiredAt != null ||
    (order.expiresAt != null && Date.parse(order.expiresAt) <= now));
}

export function orderStatusLabel(order: Order, now: number): string {
  return order.isPaid ? "Paid" : isOrderExpired(order, now) ? "Expired" : "Pending payment";
}
