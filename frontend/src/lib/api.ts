import type { Product, ProductDraft, Review, ShippingAddress, User } from "@/lib/types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1";

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
  payment_method?: string | null;
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

type ApiPageMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
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

async function request<T>(path: string, init: RequestInit): Promise<ApiResult<T>> {
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

function transformData(value: unknown): unknown {
  if (isApiUser(value)) return toUser(value);
  if (isApiProduct(value)) return toProduct(value);
  if (isApiReview(value)) return toReview(value);
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

function isApiProduct(value: unknown): value is ApiProduct {
  return Boolean(value) && typeof value === "object" && "slug" in (value as Record<string, unknown>) && "num_reviews" in (value as Record<string, unknown>);
}

function isApiReview(value: unknown): value is ApiReview {
  return Boolean(value) && typeof value === "object" && "product_id" in (value as Record<string, unknown>) && "is_verified_purchase" in (value as Record<string, unknown>);
}

function isApiPagedProducts(value: unknown): value is ApiPaged<ApiProduct> {
  return Boolean(value) && typeof value === "object" && "items" in (value as Record<string, unknown>) && "meta" in (value as Record<string, unknown>);
}

function toUser(user: ApiUser): User {
  return {
    id: user.id,
    name: user.name,
    email: user.email,
    role: user.role,
    paymentMethod: user.payment_method ?? undefined,
    address: user.address ? toShippingAddress(user.address) : undefined,
    createdAt: new Date().toISOString(),
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
