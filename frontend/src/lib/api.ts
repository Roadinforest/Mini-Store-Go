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

export type ChatStreamChunk = {
  type: "partial" | "complete" | "navigation" | "error" | "tool_call" | "thinking";
  content?: string;
  url?: string;
  message?: string;
  toolName?: string;
};

type ApiEnvelope<T> = {
  code: string;
  message: string;
  data?: T;
  details?: unknown;
};

type ApiUser = {
  id: string;
  name: string;
  email: string;
  role: "admin" | "user";
  image?: string | null;
  payment_method?: string | null;
  created_at?: string;
  address?: {
    full_name: string;
    street_address: string;
    city: string;
    postal_code: string;
    country: string;
  } | null;
};

type ApiProduct = {
  id: string;
  name: string;
  slug: string;
  category: string;
  images: string[];
  brand: string;
  description: string;
  stock: number;
  price: number;
  rating: number;
  num_reviews: number;
  is_featured: boolean;
  banner?: string | null;
  created_at: string;
};

type ApiReview = {
  id: string;
  user_id: string;
  product_id: string;
  rating: number;
  title: string;
  description: string;
  is_verified_purchase: boolean;
  created_at: string;
  user?: {
    id: string;
    name: string;
  };
  product?: {
    id: string;
    name: string;
    slug: string;
    image?: string;
  };
};

type ApiCategoryCount = {
  category: string;
  count: number;
};

type ApiCartItem = {
  product_id: string;
  name: string;
  slug: string;
  qty: number;
  image: string;
  price: number;
};

type ApiCart = {
  id?: string;
  user_id?: string | null;
  session_cart_id: string;
  items: ApiCartItem[];
  items_price: number;
  shipping_price: number;
  tax_price: number;
  total_price: number;
  created_at?: string;
};

type ApiOrderItem = ApiCartItem;

type ApiOrder = {
  id: string;
  user_id: string;
  shipping_address: {
    full_name: string;
    street_address: string;
    city: string;
    postal_code: string;
    country: string;
  };
  payment_method: string;
  items_price: number;
  shipping_price: number;
  tax_price: number;
  total_price: number;
  is_paid: boolean;
  paid_at?: string | null;
  is_delivered: boolean;
  delivered_at?: string | null;
  created_at: string;
  order_items: ApiOrderItem[];
  user?: {
    id: string;
    name: string;
    email: string;
  };
};

type ApiPageMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

type ApiAdminOverview = {
  order_count: number;
  product_count: number;
  user_count: number;
  total_sales: number;
};

type ApiPaged<T> = {
  items: T[];
  meta: ApiPageMeta;
};

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

export async function sendChat(messages: ChatMessage[]): Promise<ApiResult<ChatMessage>> {
  return request<ChatMessage>("/ai/chat", {
    method: "POST",
    body: JSON.stringify({ messages }),
  });
}

export async function createChatStream(messages: ChatMessage[]): Promise<Response> {
  return fetch(`${API_BASE_URL}/ai/chat/stream`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ messages }),
  });
}

export async function signIn(payload: { email: string; password: string }): Promise<ApiResult<User>> {
  return request<User>("/auth/sign-in", {
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
  return request<User>("/auth/sign-up", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function signOut(): Promise<ApiResult<{ signed_out: boolean }>> {
  return request<{ signed_out: boolean }>("/auth/sign-out", {
    method: "POST",
  });
}

export async function getCurrentUser(): Promise<ApiResult<User>> {
  return request<User>("/auth/me", {
    method: "GET",
  });
}

export async function refreshAuth(): Promise<ApiResult<User>> {
  return request<User>("/auth/refresh", {
    method: "POST",
  }, { skipAuthRefresh: true });
}

export async function updateProfile(payload: {
  name: string;
  email: string;
}): Promise<ApiResult<User>> {
  return request<User>("/users/me/profile", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function updateAddress(payload: ShippingAddress): Promise<ApiResult<User>> {
  return request<User>("/users/me/address", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function updatePaymentMethod(payload: {
  type: string;
}): Promise<ApiResult<User>> {
  return request<User>("/users/me/payment-method", {
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
  return request<CatalogPage<Product>>(`/products?${search.toString()}`, {
    method: "GET",
  });
}

export async function getLatestProducts(limit = 6): Promise<ApiResult<Product[]>> {
  return request<Product[]>(`/products/latest?limit=${limit}`, { method: "GET" });
}

export async function getFeaturedProducts(limit = 4): Promise<ApiResult<Product[]>> {
  return request<Product[]>(`/products/featured?limit=${limit}`, { method: "GET" });
}

export async function getProductCategories(): Promise<ApiResult<ApiCategoryCount[]>> {
  return request<ApiCategoryCount[]>("/products/categories", { method: "GET" });
}

export async function getProductBySlug(slug: string): Promise<ApiResult<Product>> {
  return request<Product>(`/products/slug/${slug}`, { method: "GET" });
}

export async function getProductByID(id: string): Promise<ApiResult<Product>> {
  return request<Product>(`/products/${id}`, { method: "GET" });
}

export async function getProductReviews(productID: string): Promise<ApiResult<Review[]>> {
  return request<Review[]>(`/reviews/product/${productID}`, { method: "GET" });
}

export async function getMyReview(productID: string): Promise<ApiResult<Review>> {
  return request<Review>(`/reviews/mine?product_id=${encodeURIComponent(productID)}`, { method: "GET" });
}

export async function upsertReview(payload: {
  product_id: string;
  rating: number;
  title: string;
  description: string;
}): Promise<ApiResult<Review>> {
  return request<Review>("/reviews", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function getAdminProducts(params: { page?: number; limit?: number } = {}): Promise<ApiResult<CatalogPage<Product>>> {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.limit) search.set("limit", String(params.limit));
  const query = search.toString();
  return request<CatalogPage<Product>>(`/admin/products${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function getAdminOverview(): Promise<ApiResult<AdminOverview>> {
  return request<AdminOverview>("/admin/overview", { method: "GET" });
}

export async function getAdminUsers(params: { page?: number; limit?: number; q?: string } = {}): Promise<ApiResult<CatalogPage<User>>> {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.limit) search.set("limit", String(params.limit));
  if (params.q) search.set("q", params.q);
  const query = search.toString();
  return request<CatalogPage<User>>(`/admin/users${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function updateAdminUser(userID: string, payload: {
  name: string;
  email: string;
  role: "admin" | "user";
}): Promise<ApiResult<User>> {
  return request<User>(`/admin/users/${userID}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function deleteAdminUser(userID: string): Promise<ApiResult<{ deleted: boolean }>> {
  return request<{ deleted: boolean }>(`/admin/users/${userID}`, {
    method: "DELETE",
  });
}

export async function createProduct(payload: ProductDraft): Promise<ApiResult<Product>> {
  return request<Product>("/admin/products", {
    method: "POST",
    body: JSON.stringify(toProductPayload(payload)),
  });
}

export async function deleteProduct(productID: string): Promise<ApiResult<{ deleted: boolean }>> {
  return request<{ deleted: boolean }>(`/admin/products/${productID}`, {
    method: "DELETE",
  });
}

export async function getCart(): Promise<ApiResult<Cart>> {
  return request<Cart>("/cart", { method: "GET" });
}

export async function addCartItem(productID: string): Promise<ApiResult<Cart>> {
  return request<Cart>("/cart/items", {
    method: "POST",
    body: JSON.stringify({ product_id: productID }),
  });
}

export async function removeCartItem(productID: string): Promise<ApiResult<Cart>> {
  return request<Cart>(`/cart/items/${productID}`, {
    method: "DELETE",
  });
}

export async function createOrder(): Promise<ApiResult<Order>> {
  return request<Order>("/orders", {
    method: "POST",
  });
}

export async function getMyOrders(): Promise<ApiResult<CatalogPage<Order>>> {
  return request<CatalogPage<Order>>("/orders/mine", { method: "GET" });
}

export async function getOrderByID(orderID: string): Promise<ApiResult<Order>> {
  return request<Order>(`/orders/${orderID}`, { method: "GET" });
}

export async function getAdminOrders(): Promise<ApiResult<CatalogPage<Order>>> {
  return request<CatalogPage<Order>>("/admin/orders", { method: "GET" });
}

export async function markOrderPaid(orderID: string): Promise<ApiResult<Order>> {
  return request<Order>(`/admin/orders/${orderID}/pay`, { method: "PUT" });
}

export async function markOrderDelivered(orderID: string): Promise<ApiResult<Order>> {
  return request<Order>(`/admin/orders/${orderID}/deliver`, { method: "PUT" });
}

async function request<T>(path: string, init: RequestInit, options: RequestOptions = {}): Promise<ApiResult<T>> {
  try {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(init.headers ?? {}),
      },
    });

    const payload = (await response.json()) as ApiEnvelope<unknown>;
    if (!response.ok) {
      if (response.status === 401 && shouldRefreshAuth(path, options)) {
        const refreshResult = await refreshAuth();
        if (refreshResult.success) {
          return request<T>(path, init, { ...options, skipAuthRefresh: true });
        }
      }

      return {
        success: false,
        message: payload.message || "Request failed.",
        details: payload.details,
      };
    }

    return {
      success: true,
      message: payload.message || "success",
      data: transformData(payload.data) as T,
    };
  } catch (error) {
    return {
      success: false,
      message: error instanceof Error ? error.message : "Network error.",
    };
  }
}

function shouldRefreshAuth(path: string, options: RequestOptions): boolean {
  if (options.skipAuthRefresh) return false;
  return !["/auth/sign-in", "/auth/sign-up", "/auth/sign-out", "/auth/refresh"].includes(path);
}

function transformData(value: unknown): unknown {
  if (isApiAdminOverview(value)) return toAdminOverview(value);
  if (isApiUser(value)) return toUser(value);
  if (isApiProduct(value)) return toProduct(value);
  if (isApiReview(value)) return toReview(value);
  if (isApiCart(value)) return toCart(value);
  if (isApiOrder(value)) return toOrder(value);
  if (isApiPagedOrders(value)) return toOrderCatalogPage(value);
  if (isApiPagedUsers(value)) return toUserCatalogPage(value);
  if (isApiPagedProducts(value)) return toCatalogPage(value);
  if (Array.isArray(value)) {
    if (value.every(isApiProduct)) return value.map(toProduct);
    if (value.every(isApiReview)) return value.map(toReview);
  }
  return value;
}

function isApiUser(value: unknown): value is ApiUser {
  return Boolean(value) && typeof value === "object" && "id" in (value as Record<string, unknown>) && "email" in (value as Record<string, unknown>);
}

function isApiAdminOverview(value: unknown): value is ApiAdminOverview {
  return Boolean(value) && typeof value === "object" && "order_count" in (value as Record<string, unknown>) && "total_sales" in (value as Record<string, unknown>);
}

function isApiProduct(value: unknown): value is ApiProduct {
  return Boolean(value) && typeof value === "object" && "slug" in (value as Record<string, unknown>) && "num_reviews" in (value as Record<string, unknown>);
}

function isApiReview(value: unknown): value is ApiReview {
  return Boolean(value) && typeof value === "object" && "product_id" in (value as Record<string, unknown>) && "is_verified_purchase" in (value as Record<string, unknown>);
}

function isApiPagedProducts(value: unknown): value is ApiPaged<ApiProduct> {
  if (!Boolean(value) || typeof value !== "object" || !("items" in (value as Record<string, unknown>)) || !("meta" in (value as Record<string, unknown>))) {
    return false;
  }
  const items = (value as { items?: unknown[] }).items;
  return Array.isArray(items) && (items.length === 0 || items.every(isApiProduct));
}

function isApiPagedUsers(value: unknown): value is ApiPaged<ApiUser> {
  if (!Boolean(value) || typeof value !== "object" || !("items" in (value as Record<string, unknown>)) || !("meta" in (value as Record<string, unknown>))) {
    return false;
  }
  const items = (value as { items?: unknown[] }).items;
  return Array.isArray(items) && (items.length === 0 || items.every(isApiUser));
}

function isApiCart(value: unknown): value is ApiCart {
  return Boolean(value) && typeof value === "object" && "session_cart_id" in (value as Record<string, unknown>) && "items_price" in (value as Record<string, unknown>);
}

function isApiOrder(value: unknown): value is ApiOrder {
  return Boolean(value) && typeof value === "object" && "shipping_address" in (value as Record<string, unknown>) && "order_items" in (value as Record<string, unknown>);
}

function isApiPagedOrders(value: unknown): value is ApiPaged<ApiOrder> {
  if (!Boolean(value) || typeof value !== "object" || !("items" in (value as Record<string, unknown>)) || !("meta" in (value as Record<string, unknown>))) {
    return false;
  }
  const items = (value as { items?: unknown[] }).items;
  return Array.isArray(items) && (items.length === 0 || items.every(isApiOrder));
}

function toUser(user: ApiUser): User {
  return {
    id: user.id,
    name: user.name,
    email: user.email,
    role: user.role,
    paymentMethod: user.payment_method ?? undefined,
    address: user.address ? toShippingAddress(user.address) : undefined,
    createdAt: user.created_at ?? new Date().toISOString(),
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
    rating: Number(product.rating ?? 0).toFixed(2),
    num_reviews: Number(product.numReviews ?? 0),
  };
}
