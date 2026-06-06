import type { Cart, CartItem, Product } from "@/lib/types";

export const APP_NAME = "Mini Store";
export const TAX_RATE = 0.15;
export const SHIPPING_PRICE = 10;
export const FREE_SHIPPING_BAR = 100;
export const PAYMENT_METHODS = ["PayPal", "Stripe", "CashOnDelivery"];

export function formatCurrency(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(value);
}

export function formatDate(value: string | null) {
  if (!value) return "N/A";
  return new Date(value).toLocaleString();
}

export function slugify(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function calcCart(items: CartItem[]): Cart {
  const itemsPrice = round2(items.reduce((sum, item) => sum + item.price * item.qty, 0));
  const shippingPrice = round2(itemsPrice > FREE_SHIPPING_BAR ? 0 : SHIPPING_PRICE);
  const taxPrice = round2(itemsPrice * TAX_RATE);
  const totalPrice = round2(itemsPrice + shippingPrice + taxPrice);
  return { items, itemsPrice, shippingPrice, taxPrice, totalPrice };
}

export function round2(value: number) {
  return Math.round((value + Number.EPSILON) * 100) / 100;
}

export function getCategoryCounts(products: Product[]) {
  const counts = new Map<string, number>();
  for (const product of products) {
    counts.set(product.category, (counts.get(product.category) ?? 0) + 1);
  }
  return Array.from(counts.entries()).map(([category, count]) => ({ category, count }));
}

export function getAverageRatingForProduct(productId: string, reviews: Array<{ productId: string; rating: number }>) {
  const productReviews = reviews.filter((review) => review.productId === productId);
  if (productReviews.length === 0) return { rating: 0, numReviews: 0 };
  const total = productReviews.reduce((sum, review) => sum + review.rating, 0);
  return {
    rating: round2(total / productReviews.length),
    numReviews: productReviews.length,
  };
}
