import type { ShippingAddress, User } from "@/lib/types";

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

    const payload = (await response.json()) as ApiEnvelope<ApiUser | T>;
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
      data: isApiUser(payload.data) ? (toUser(payload.data) as T) : payload.data,
    };
  } catch (error) {
    return {
      success: false,
      message: error instanceof Error ? error.message : "Network error.",
    };
  }
}

function isApiUser(value: unknown): value is ApiUser {
  return Boolean(value) && typeof value === "object" && "id" in (value as Record<string, unknown>) && "email" in (value as Record<string, unknown>);
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

function toShippingAddress(address: NonNullable<ApiUser["address"]>): ShippingAddress {
  return {
    fullName: address.full_name,
    streetAddress: address.street_address,
    city: address.city,
    postalCode: address.postal_code,
    country: address.country,
  };
}
