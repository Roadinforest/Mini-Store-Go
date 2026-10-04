import { z } from "zod";
import type { AdminOverview, Cart, CartItem, Order, Product, ProductDraft, Review, ShippingAddress, User } from "@/lib/types";

const API_BASE_URL = import.meta.env.DEV
  ? (import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1")
  : "/api/v1";

export type ChatMessage = {
  role: "user" | "assistant" | "system";
  content: string;
  url?: string;
  messageType?: "normal" | "thinking" | "tool_call" | "navigation";
  toolName?: string;
  toolCalls?: Array<{
    toolName: string;
    content: string;
  }>;
};


const addressSchema = z.object({
  full_name: z.string(), street_address: z.string(), city: z.string(), postal_code: z.string(), country: z.string(),
});
const apiUserSchema = z.object({
  id: z.string(), name: z.string(), email: z.string(), role: z.enum(["admin", "user"]),
  payment_method: z.string().nullish(), created_at: z.string().optional(), address: addressSchema.nullish(),
});
const apiProductSchema = z.object({
  id: z.string(), name: z.string(), slug: z.string(), category: z.string(),
  images: z.array(z.string()).nullable().transform(value => value ?? []),
  brand: z.string(), description: z.string(), stock: z.number().int().nonnegative(),
  price: z.number().nonnegative(), rating: z.number(), num_reviews: z.number().int().nonnegative(),
  is_featured: z.boolean(), banner: z.string().nullish(), created_at: z.string(),
});
const apiReviewSchema = z.object({
  id: z.string(), user_id: z.string(), product_id: z.string(), rating: z.number(),
  title: z.string(), description: z.string(), is_verified_purchase: z.boolean(), created_at: z.string(),
  user: z.object({id: z.string(), name: z.string()}).optional(),
  product: z.object({id: z.string(), name: z.string(), slug: z.string(), image: z.string().optional()}).optional(),
});
const apiCartItemSchema = z.object({
  product_id: z.string(), name: z.string(), slug: z.string(), qty: z.number().int().positive(), image: z.string(), price: z.number(),
});
const pricesSchema = z.object({items_price: z.number(), shipping_price: z.number(), tax_price: z.number(), total_price: z.number()});
const apiCartSchema = pricesSchema.extend({session_cart_id: z.string(), items: z.array(apiCartItemSchema)});
const apiOrderSchema = pricesSchema.extend({
  id: z.string(), user_id: z.string(), shipping_address: addressSchema, payment_method: z.string(),
  is_paid: z.boolean(), paid_at: z.string().nullish(), is_delivered: z.boolean(), delivered_at: z.string().nullish(),
  status: z.enum(["UNPAID", "PAID", "EXPIRED"]).optional(), expires_at: z.string().nullish(), expired_at: z.string().nullish(),
  created_at: z.string(), order_items: z.array(apiCartItemSchema),
  user: z.object({id: z.string(), name: z.string(), email: z.string()}).optional(),
});
const apiPageMetaSchema = z.object({page: z.number().int().positive(), limit: z.number().int().positive(), total: z.number().int().nonnegative(), total_pages: z.number().int().nonnegative()});
function paged<S extends z.ZodType>(item: S) { return z.object({items: z.array(item), meta: apiPageMetaSchema}); }
const apiAdminOverviewSchema = z.object({order_count: z.number(), product_count: z.number(), user_count: z.number(), total_sales: z.number()});
const envelopeSchema = z.object({code: z.string(), message: z.string(), data: z.unknown().optional(), details: z.unknown().optional()});
const chatSchema = z.object({
  role: z.enum(["assistant", "user", "system"]), content: z.string(), url: z.string().optional(),
  messageType: z.enum(["normal", "thinking", "tool_call", "navigation"]).optional(), toolName: z.string().optional(),
  toolCalls: z.array(z.object({toolName: z.string(), content: z.string()})).optional(),
});
export const chatStreamSchema = z.object({
  type: z.enum(["partial", "complete", "navigation", "error", "tool_call", "tool_result", "thinking"]),
  content: z.string().optional(), url: z.string().optional(), message: z.string().optional(), toolName: z.string().optional(),
});
export type ChatStreamChunk = z.infer<typeof chatStreamSchema>;
type ApiUser = z.infer<typeof apiUserSchema>;
type ApiProduct = z.infer<typeof apiProductSchema>;
type ApiReview = z.infer<typeof apiReviewSchema>;
type ApiCartItem = z.infer<typeof apiCartItemSchema>;
type ApiCart = z.infer<typeof apiCartSchema>;
type ApiOrder = z.infer<typeof apiOrderSchema>;
type ApiAdminOverview = z.infer<typeof apiAdminOverviewSchema>;
type ApiPaged<T> = {items: T[]; meta: z.infer<typeof apiPageMetaSchema>};
type ApiCategoryCount = {category: string; count: number};

const userSchema = apiUserSchema.transform(toUser);
const productSchema = apiProductSchema.transform(toProduct);
const reviewSchema = apiReviewSchema.transform(toReview);
const cartSchema = apiCartSchema.transform(toCart);
const orderSchema = apiOrderSchema.transform(toOrder);
const productsPageSchema = paged(apiProductSchema).transform(toCatalogPage);
const usersPageSchema = paged(apiUserSchema).transform(toUserCatalogPage);
const ordersPageSchema = paged(apiOrderSchema).transform(toOrderCatalogPage);
const overviewSchema = apiAdminOverviewSchema.transform(toAdminOverview);
const categoriesSchema = z.array(z.object({category: z.string(), count: z.number().int().nonnegative()}));
const deletedSchema = z.object({deleted: z.boolean()});
const signOutSchema = z.object({signed_out: z.boolean()});

export type CatalogPage<T> = {
  items: T[];
  meta: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
};

export type ApiResult<T> = {
  success: boolean;
  message: string;
  data?: T;
  details?: unknown;
};

type RequestOptions = {
  skipAuthRefresh?: boolean;
};

export async function sendChat(messages: ChatMessage[], signal?: AbortSignal): Promise<ApiResult<ChatMessage>> {
  return request(chatSchema, "/ai/chat", {
    signal,
    method: "POST",
    body: JSON.stringify({ messages }),
  });
}

export async function createChatStream(messages: ChatMessage[], signal?: AbortSignal): Promise<Response> {
  return fetch(`${API_BASE_URL}/ai/chat/stream`, {
    signal,
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ messages }),
  });
}

export async function signIn(payload: { email: string; password: string }): Promise<ApiResult<User>> {
  return request(userSchema, "/auth/sign-in", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function signUp(payload: {
  name: string;
  email: string;
  password: string;
  confirm_password: string;
}): Promise<ApiResult<User>> {
  return request(userSchema, "/auth/sign-up", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function signOut(): Promise<ApiResult<{ signed_out: boolean }>> {
  return request(signOutSchema, "/auth/sign-out", {
    method: "POST",
  });
}

export async function getCurrentUser(): Promise<ApiResult<User>> {
  return request(userSchema, "/auth/me", {
    method: "GET",
  });
}

export async function refreshAuth(): Promise<ApiResult<User>> {
  return request(userSchema, "/auth/refresh", {
    method: "POST",
  }, { skipAuthRefresh: true });
}

export async function updateProfile(payload: {
  name: string;
  email: string;
}): Promise<ApiResult<User>> {
  return request(userSchema, "/users/me/profile", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function updateAddress(payload: ShippingAddress): Promise<ApiResult<User>> {
  return request(userSchema, "/users/me/address", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function updatePaymentMethod(payload: {
  type: string;
}): Promise<ApiResult<User>> {
  return request(userSchema, "/users/me/payment-method", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function getProducts(params: {
  page?: number;
  limit?: number;
  q?: string;
  category?: string;
  price?: string;
  rating?: string;
  sort?: string;
} = {}): Promise<ApiResult<CatalogPage<Product>>> {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && String(value) !== "") {
      search.set(key, String(value));
    }
  }
  return request(productsPageSchema, `/products?${search.toString()}`, {
    method: "GET",
  });
}

export async function getLatestProducts(limit = 6): Promise<ApiResult<Product[]>> {
  return request(z.array(productSchema), `/products/latest?limit=${limit}`, { method: "GET" });
}

export async function getFeaturedProducts(limit = 4): Promise<ApiResult<Product[]>> {
  return request(z.array(productSchema), `/products/featured?limit=${limit}`, { method: "GET" });
}

export async function getProductCategories(): Promise<ApiResult<ApiCategoryCount[]>> {
  return request(categoriesSchema, "/products/categories", { method: "GET" });
}

export async function getProductBySlug(slug: string): Promise<ApiResult<Product>> {
  return request(productSchema, `/products/slug/${slug}`, { method: "GET" });
}

export async function getProductByID(id: string): Promise<ApiResult<Product>> {
  return request(productSchema, `/products/${id}`, { method: "GET" });
}

export async function getProductReviews(productID: string): Promise<ApiResult<Review[]>> {
  return request(z.array(reviewSchema), `/reviews/product/${productID}`, { method: "GET" });
}

export async function getMyReview(productID: string): Promise<ApiResult<Review>> {
  return request(reviewSchema, `/reviews/mine?product_id=${encodeURIComponent(productID)}`, { method: "GET" });
}

export async function upsertReview(payload: {
  product_id: string;
  rating: number;
  title: string;
  description: string;
}): Promise<ApiResult<Review>> {
  return request(reviewSchema, "/reviews", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function getAdminProducts(params: { page?: number; limit?: number } = {}): Promise<ApiResult<CatalogPage<Product>>> {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.limit) search.set("limit", String(params.limit));
  const query = search.toString();
  return request(productsPageSchema, `/admin/products${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function getAdminOverview(): Promise<ApiResult<AdminOverview>> {
  return request(overviewSchema, "/admin/overview", { method: "GET" });
}

export async function getAdminUsers(params: { page?: number; limit?: number; q?: string } = {}): Promise<ApiResult<CatalogPage<User>>> {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.limit) search.set("limit", String(params.limit));
  if (params.q) search.set("q", params.q);
  const query = search.toString();
  return request(usersPageSchema, `/admin/users${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function updateAdminUser(userID: string, payload: {
  name: string;
  email: string;
  role: "admin" | "user";
}): Promise<ApiResult<User>> {
  return request(userSchema, `/admin/users/${userID}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function deleteAdminUser(userID: string): Promise<ApiResult<{ deleted: boolean }>> {
  return request(deletedSchema, `/admin/users/${userID}`, {
    method: "DELETE",
  });
}

export async function createProduct(payload: ProductDraft): Promise<ApiResult<Product>> {
  return request(productSchema, "/admin/products", {
    method: "POST",
    body: JSON.stringify(toProductPayload(payload)),
  });
}

export async function deleteProduct(productID: string): Promise<ApiResult<{ deleted: boolean }>> {
  return request(deletedSchema, `/admin/products/${productID}`, {
    method: "DELETE",
  });
}

export async function getCart(): Promise<ApiResult<Cart>> {
  return request(cartSchema, "/cart", { method: "GET" });
}

export async function addCartItem(productID: string): Promise<ApiResult<Cart>> {
  return request(cartSchema, "/cart/items", {
    method: "POST",
    body: JSON.stringify({ product_id: productID }),
  });
}

export async function removeCartItem(productID: string): Promise<ApiResult<Cart>> {
  return request(cartSchema, `/cart/items/${productID}`, {
    method: "DELETE",
  });
}

export async function createOrder(): Promise<ApiResult<Order>> {
  return request(orderSchema, "/orders", {
    method: "POST",
  });
}

export async function getMyOrders(): Promise<ApiResult<CatalogPage<Order>>> {
  return request(ordersPageSchema, "/orders/mine", { method: "GET" });
}

export async function getOrderByID(orderID: string): Promise<ApiResult<Order>> {
  return request(orderSchema, `/orders/${orderID}`, { method: "GET" });
}

export async function getAdminOrders(): Promise<ApiResult<CatalogPage<Order>>> {
  return request(ordersPageSchema, "/admin/orders", { method: "GET" });
}

export async function markOrderPaid(orderID: string): Promise<ApiResult<Order>> {
  return request(orderSchema, `/admin/orders/${orderID}/pay`, { method: "PUT" });
}

export async function markOrderDelivered(orderID: string): Promise<ApiResult<Order>> {
  return request(orderSchema, `/admin/orders/${orderID}/deliver`, { method: "PUT" });
}

async function request<S extends z.ZodType>(schema: S, path: string, init: RequestInit, options: RequestOptions = {}): Promise<ApiResult<z.output<S>>> {
  try {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(init.headers ?? {}),
      },
    });

    const payload = envelopeSchema.parse(await response.json());
    if (!response.ok) {
      if (response.status === 401 && shouldRefreshAuth(path, options)) {
        const refreshResult = await refreshOnce();
        if (refreshResult.success) {
          return request(schema, path, init, { ...options, skipAuthRefresh: true });
        }
      }

      return {
        success: false,
        message: payload.message || "Request failed.",
        details: payload.details,
      };
    }

    if (payload.code !== "OK") return { success: false, message: payload.message || "Request failed.", details: payload.details };
    return {
      success: true,
      message: payload.message || "success",
      data: schema.parse(payload.data),
    };
  } catch (error) {
    return {
      success: false,
      message: error instanceof z.ZodError ? "Invalid API response." : error instanceof Error ? error.message : "Network error.",
    };
  }
}

function shouldRefreshAuth(path: string, options: RequestOptions): boolean {
  if (options.skipAuthRefresh) return false;
  return !["/auth/sign-in", "/auth/sign-up", "/auth/sign-out", "/auth/refresh"].includes(path);
}

let pendingRefresh: Promise<ApiResult<User>> | undefined;
function refreshOnce(): Promise<ApiResult<User>> {
  if (!pendingRefresh) pendingRefresh = refreshAuth().finally(() => { pendingRefresh = undefined; });
  return pendingRefresh;
}

function toUser(user: ApiUser): User {
  return {
    id: user.id,
    name: user.name,
    email: user.email,
    role: user.role,
    paymentMethod: user.payment_method ?? undefined,
    address: user.address ? toShippingAddress(user.address) : undefined,
    createdAt: user.created_at ?? "",
  };
}

function toAdminOverview(overview: ApiAdminOverview): AdminOverview {
  return {
    orderCount: overview.order_count,
    productCount: overview.product_count,
    userCount: overview.user_count,
    totalSales: overview.total_sales,
  };
}

function toProduct(product: ApiProduct): Product {
  return {
    id: product.id,
    name: product.name,
    slug: product.slug,
    category: product.category,
    images: product.images,
    brand: product.brand,
    description: product.description,
    stock: product.stock,
    price: product.price,
    rating: product.rating,
    numReviews: product.num_reviews,
    isFeatured: product.is_featured,
    banner: product.banner ?? null,
    createdAt: product.created_at,
  };
}

function toReview(review: ApiReview): Review {
  return {
    id: review.id,
    userId: review.user_id,
    productId: review.product_id,
    rating: review.rating,
    title: review.title,
    description: review.description,
    isVerifiedPurchase: review.is_verified_purchase,
    createdAt: review.created_at,
    user: review.user,
    product: review.product,
  };
}

function toCatalogPage(page: ApiPaged<ApiProduct>): CatalogPage<Product> {
  return {
    items: page.items.map(toProduct),
    meta: {
      page: page.meta.page,
      limit: page.meta.limit,
      total: page.meta.total,
      totalPages: page.meta.total_pages,
    },
  };
}

function toUserCatalogPage(page: ApiPaged<ApiUser>): CatalogPage<User> {
  return {
    items: page.items.map(toUser),
    meta: {
      page: page.meta.page,
      limit: page.meta.limit,
      total: page.meta.total,
      totalPages: page.meta.total_pages,
    },
  };
}

function toCart(cart: ApiCart): Cart {
  return {
    items: cart.items.map(toCartItem),
    itemsPrice: cart.items_price,
    shippingPrice: cart.shipping_price,
    taxPrice: cart.tax_price,
    totalPrice: cart.total_price,
  };
}

function toCartItem(item: ApiCartItem): CartItem {
  return {
    productId: item.product_id,
    name: item.name,
    slug: item.slug,
    qty: item.qty,
    image: item.image,
    price: item.price,
  };
}

function toOrder(order: ApiOrder): Order {
  return {
    id: order.id,
    userId: order.user_id,
    shippingAddress: toShippingAddress(order.shipping_address),
    paymentMethod: order.payment_method,
    itemsPrice: order.items_price,
    shippingPrice: order.shipping_price,
    taxPrice: order.tax_price,
    totalPrice: order.total_price,
    isPaid: order.is_paid,
    status: order.status ?? (order.is_paid ? "PAID" : "UNPAID"),
    expiresAt: order.expires_at ?? null,
    expiredAt: order.expired_at ?? null,
    paidAt: order.paid_at ?? null,
    isDelivered: order.is_delivered,
    deliveredAt: order.delivered_at ?? null,
    createdAt: order.created_at,
    orderitems: order.order_items.map(toCartItem),
    user: order.user,
  };
}

function toOrderCatalogPage(page: ApiPaged<ApiOrder>): CatalogPage<Order> {
  return {
    items: page.items.map(toOrder),
    meta: {
      page: page.meta.page,
      limit: page.meta.limit,
      total: page.meta.total,
      totalPages: page.meta.total_pages,
    },
  };
}

function toShippingAddress(address: NonNullable<ApiUser["address"]>): ShippingAddress {
  return {
    fullName: address.full_name,
    streetAddress: address.street_address,
    city: address.city,
    postalCode: address.postal_code,
    country: address.country,
  };
}

function toProductPayload(product: ProductDraft) {
  return {
    name: product.name,
    slug: product.slug,
    category: product.category,
    brand: product.brand,
    description: product.description,
    stock: Number(product.stock),
    images: product.images,
    is_featured: product.isFeatured,
    banner: product.banner,
    price: Number(product.price).toFixed(2),
  };
}
