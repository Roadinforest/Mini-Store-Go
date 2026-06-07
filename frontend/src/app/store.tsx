import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useState,
  type PropsWithChildren,
} from "react";
import * as authApi from "@/lib/api";
import type {
  AppState,
  Order,
  Product,
  ProductDraft,
  Review,
  ShippingAddress,
  User,
} from "@/lib/types";
import { calcCart, getAverageRatingForProduct, slugify } from "@/lib/utils";
import { createInitialState } from "@/mock/data";

const STORAGE_KEY = "mini-store-go-mock-state";

type AuthPayload = { email: string; password: string };
type SignUpPayload = { name: string; email: string; password: string };
type ProfilePayload = { name: string; email: string };
type ReviewPayload = { rating: number; title: string; description: string };
type Result = { success: boolean; message: string };

type StoreContextValue = {
  state: AppState;
  currentUser: User | null;
  authReady: boolean;
  signIn: (payload: AuthPayload) => Promise<Result>;
  signUp: (payload: SignUpPayload) => Promise<Result>;
  signOut: () => Promise<void>;
  addToCart: (productId: string) => Promise<Result>;
  removeFromCart: (productId: string) => Promise<Result>;
  setShippingAddress: (address: ShippingAddress) => Promise<Result>;
  setPaymentMethod: (method: string) => Promise<Result>;
  updateProfile: (payload: ProfilePayload) => Promise<Result>;
  placeOrder: () => Promise<{ success: boolean; message: string; orderId?: string }>;
  markOrderPaid: (orderId: string) => Promise<{ success: boolean; message: string }>;
  markOrderDelivered: (orderId: string) => Promise<{ success: boolean; message: string }>;
  saveProduct: (draft: ProductDraft) => void;
  deleteProduct: (productId: string) => void;
  updateUser: (userId: string, payload: Pick<User, "name" | "role">) => void;
  deleteUser: (userId: string) => void;
  upsertReview: (productId: string, payload: ReviewPayload) => Result;
  syncProducts: (products: Product[]) => void;
  syncReviews: (reviews: Review[]) => void;
};

type Action =
  | { type: "SET_STATE"; payload: AppState }
  | { type: "SIGN_IN"; payload: string }
  | { type: "SIGN_OUT" }
  | { type: "SET_CART"; payload: AppState["cart"] }
  | { type: "SET_USERS"; payload: User[] }
  | { type: "SET_PRODUCTS"; payload: Product[] }
  | { type: "SET_ORDERS"; payload: Order[] }
  | { type: "SET_REVIEWS"; payload: Review[] };

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case "SET_STATE":
      return action.payload;
    case "SIGN_IN":
      return { ...state, currentUserId: action.payload };
    case "SIGN_OUT":
      return { ...state, currentUserId: null };
    case "SET_CART":
      return { ...state, cart: action.payload };
    case "SET_USERS":
      return { ...state, users: action.payload };
    case "SET_PRODUCTS":
      return { ...state, products: action.payload };
    case "SET_ORDERS":
      return { ...state, orders: action.payload };
    case "SET_REVIEWS":
      return { ...state, reviews: action.payload };
    default:
      return state;
  }
}

function loadInitialState() {
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) return createInitialState();

  try {
    return JSON.parse(raw) as AppState;
  } catch {
    return createInitialState();
  }
}

const StoreContext = createContext<StoreContextValue | null>(null);

export function StoreProvider({ children }: PropsWithChildren) {
  const [state, dispatch] = useReducer(reducer, undefined, loadInitialState);
  const [authReady, setAuthReady] = useState(false);

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  }, [state]);

  useEffect(() => {
    let cancelled = false;

    async function bootstrapAuth() {
      const [authResult, cartResult] = await Promise.all([
        authApi.getCurrentUser(),
        authApi.getCart(),
      ]);
      if (cancelled) return;

      if (authResult.success && authResult.data) {
        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, authResult.data) });
        dispatch({ type: "SIGN_IN", payload: authResult.data.id });
      } else {
        dispatch({ type: "SIGN_OUT" });
      }
      if (cartResult.success && cartResult.data) {
        dispatch({ type: "SET_CART", payload: cartResult.data });
      }

      setAuthReady(true);
    }

    void bootstrapAuth();

    return () => {
      cancelled = true;
    };
  }, []);

  const currentUser = useMemo(
    () => state.users.find((user) => user.id === state.currentUserId) ?? null,
    [state.currentUserId, state.users],
  );

  const value = useMemo<StoreContextValue>(() => {
    return {
      state,
      currentUser,
      authReady,
      async signIn(payload) {
        const result = await authApi.signIn(payload);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }

        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, result.data) });
        dispatch({ type: "SIGN_IN", payload: result.data.id });
        return { success: true, message: "Signed in." };
      },
      async signUp(payload) {
        const result = await authApi.signUp({
          ...payload,
          confirm_password: payload.password,
        });
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }

        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, result.data) });
        dispatch({ type: "SIGN_IN", payload: result.data.id });
        return { success: true, message: "Account created." };
      },
      async signOut() {
        await authApi.signOut();
        dispatch({ type: "SIGN_OUT" });
      },
      async addToCart(productId) {
        const product = state.products.find((item) => item.id === productId);
        if (!product) return { success: false, message: "Product not found." };
        const result = await authApi.addCartItem(productId);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_CART", payload: result.data });
        return { success: true, message: `${product.name} added to cart.` };
      },
      async removeFromCart(productId) {
        const result = await authApi.removeCartItem(productId);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_CART", payload: result.data });
        return { success: true, message: "Cart updated." };
      },
      async setShippingAddress(address) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }
        const result = await authApi.updateAddress(address);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, result.data) });
        return { success: true, message: "Shipping address saved." };
      },
      async setPaymentMethod(method) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }
        const result = await authApi.updatePaymentMethod({ type: method });
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, result.data) });
        return { success: true, message: "Payment method saved." };
      },
      async updateProfile(payload) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }
        const result = await authApi.updateProfile(payload);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, result.data) });
        return { success: true, message: "Profile updated." };
      },
      async placeOrder() {
        const result = await authApi.createOrder();
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_ORDERS", payload: [result.data, ...state.orders] });
        dispatch({ type: "SET_CART", payload: calcCart([]) });
        return { success: true, message: "Order created.", orderId: result.data.id };
      },
      async markOrderPaid(orderId) {
        const result = await authApi.markOrderPaid(orderId);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({
          type: "SET_ORDERS",
          payload: state.orders.map((item) => (item.id === orderId ? result.data! : item)),
        });
        return { success: true, message: "Order marked as paid." };
      },
      async markOrderDelivered(orderId) {
        const result = await authApi.markOrderDelivered(orderId);
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({
          type: "SET_ORDERS",
          payload: state.orders.map((item) => (item.id === orderId ? result.data! : item)),
        });
        return { success: true, message: "Order marked as delivered." };
      },
      saveProduct(draft) {
        const normalized: Product = {
          id: draft.id ?? crypto.randomUUID(),
          name: draft.name,
          slug: draft.slug || slugify(draft.name),
          category: draft.category,
          images: draft.images,
          brand: draft.brand,
          description: draft.description,
          stock: Number(draft.stock),
          price: Number(draft.price),
          rating: draft.rating ?? 0,
          numReviews: draft.numReviews ?? 0,
          isFeatured: draft.isFeatured,
          banner: draft.banner,
          createdAt: draft.createdAt ?? new Date().toISOString(),
        };
        const exists = state.products.some((product) => product.id === normalized.id);
        const nextProducts = exists
          ? state.products.map((product) =>
              product.id === normalized.id ? normalized : product,
            )
          : [normalized, ...state.products];
        dispatch({ type: "SET_PRODUCTS", payload: nextProducts });
      },
      deleteProduct(productId) {
        dispatch({
          type: "SET_PRODUCTS",
          payload: state.products.filter((product) => product.id !== productId),
        });
      },
      updateUser(userId, payload) {
        dispatch({
          type: "SET_USERS",
          payload: state.users.map((user) =>
            user.id === userId ? { ...user, ...payload } : user,
          ),
        });
      },
      deleteUser(userId) {
        dispatch({
          type: "SET_USERS",
          payload: state.users.filter((user) => user.id !== userId),
        });
      },
      upsertReview(productId, payload) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }

        const existing = state.reviews.find(
          (review) => review.productId === productId && review.userId === currentUser.id,
        );
        const nextReviews = existing
          ? state.reviews.map((review) =>
              review.id === existing.id
                ? { ...review, ...payload }
                : review,
            )
          : [
              {
                id: crypto.randomUUID(),
                userId: currentUser.id,
                productId,
                rating: payload.rating,
                title: payload.title,
                description: payload.description,
                isVerifiedPurchase: true,
                createdAt: new Date().toISOString(),
              },
              ...state.reviews,
            ];

        const nextProducts = state.products.map((product) => {
          if (product.id !== productId) return product;
          const summary = getAverageRatingForProduct(productId, nextReviews);
          return {
            ...product,
            rating: summary.rating,
            numReviews: summary.numReviews,
          };
        });

        dispatch({ type: "SET_REVIEWS", payload: nextReviews });
        dispatch({ type: "SET_PRODUCTS", payload: nextProducts });
        return { success: true, message: "Review saved." };
      },
      syncProducts(products) {
        const merged = [...state.products];
        for (const incoming of products) {
          const index = merged.findIndex((product) => product.id === incoming.id);
          if (index >= 0) {
            merged[index] = { ...merged[index], ...incoming };
          } else {
            merged.push(incoming);
          }
        }
        dispatch({ type: "SET_PRODUCTS", payload: merged });
      },
      syncReviews(reviews) {
        const merged = [...state.reviews];
        for (const incoming of reviews) {
          const index = merged.findIndex((review) => review.id === incoming.id);
          if (index >= 0) {
            merged[index] = { ...merged[index], ...incoming };
          } else {
            merged.push(incoming);
          }
        }
        dispatch({ type: "SET_REVIEWS", payload: merged });
      },
    };
  }, [authReady, currentUser, state]);

  return <StoreContext.Provider value={value}>{children}</StoreContext.Provider>;
}

export function useStore() {
  const context = useContext(StoreContext);
  if (!context) {
    throw new Error("useStore must be used within StoreProvider");
  }
  return context;
}

function upsertUser(users: User[], nextUser: User) {
  const exists = users.some((user) => user.id === nextUser.id);
  if (!exists) {
    return [...users, nextUser];
  }
  return users.map((user) => (user.id === nextUser.id ? { ...user, ...nextUser } : user));
}
