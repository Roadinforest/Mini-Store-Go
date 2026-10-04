import {
  createContext,
  useContext,
  useCallback,
  useRef,
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
  Review,
  ShippingAddress,
  User,
} from "@/lib/types";
import { calcCart } from "@/lib/utils";

type AuthPayload = { email: string; password: string };
type SignUpPayload = { name: string; email: string; password: string };
type ProfilePayload = { name: string; email: string };
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
  syncProducts: (products: Product[]) => void;
  syncReviews: (reviews: Review[]) => void;
};

type Action =
  | { type: "SIGN_IN"; payload: string }
  | { type: "SIGN_OUT" }
  | { type: "SET_CART"; payload: AppState["cart"] }
  | { type: "SET_USERS"; payload: User[] }
  | { type: "SET_PRODUCTS"; payload: Product[] }
  | { type: "SET_ORDERS"; payload: Order[] }
  | { type: "SET_REVIEWS"; payload: Review[] };

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case "SIGN_IN":
      return { ...state, users: state.users.filter(user => user.id === action.payload), orders: [], cart: calcCart([]), currentUserId: action.payload };
    case "SIGN_OUT":
      return { ...state, users: [], orders: [], cart: calcCart([]), currentUserId: null };
    case "SET_CART":
      return { ...state, cart: action.payload };
    case "SET_USERS":
      return { ...state, users: mergeByID(state.users, action.payload) };
    case "SET_PRODUCTS":
      return { ...state, products: mergeByID(state.products, action.payload) };
    case "SET_ORDERS":
      return { ...state, orders: mergeByID(state.orders, action.payload) };
    case "SET_REVIEWS":
      return { ...state, reviews: mergeByID(state.reviews, action.payload) };
    default:
      return state;
  }
}

function loadInitialState(): AppState {
  return { products: [], users: [], reviews: [], orders: [], cart: calcCart([]), currentUserId: null };
}

function mergeByID<T extends { id: string }>(current: T[], incoming: T[]): T[] {
  const merged = new Map(current.map(item => [item.id, item]));
  incoming.forEach(item => merged.set(item.id, item));
  return [...merged.values()];
}

const StoreContext = createContext<StoreContextValue | null>(null);

export function StoreProvider({ children }: PropsWithChildren) {
  const [state, dispatch] = useReducer(reducer, undefined, loadInitialState);
  const [authReady, setAuthReady] = useState(false);

  const sessionVersion = useRef(0);
  const syncProducts = useCallback((products: Product[]) => dispatch({ type: "SET_PRODUCTS", payload: products }), []);
  const syncReviews = useCallback((reviews: Review[]) => dispatch({ type: "SET_REVIEWS", payload: reviews }), []);

  useEffect(() => {
    // Remove the obsolete cache, which could contain mock passwords and roles.
    try { localStorage.removeItem("mini-store-go-mock-state"); } catch { /* Storage is optional. */ }
    let cancelled = false;
    const version = sessionVersion.current;

    async function bootstrapAuth() {
      const [authResult, cartResult] = await Promise.all([
        authApi.getCurrentUser(),
        authApi.getCart(),
      ]);
      if (cancelled || version !== sessionVersion.current) return;

      if (authResult.success && authResult.data) {
        dispatch({ type: "SET_USERS", payload: [authResult.data] });
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
        ++sessionVersion.current;
        const version = sessionVersion.current;
        const result = await authApi.signIn(payload);
        setAuthReady(true);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }

        dispatch({ type: "SET_USERS", payload: [result.data] });
        dispatch({ type: "SIGN_IN", payload: result.data.id });
        const cart = await authApi.getCart();
        if (version === sessionVersion.current && cart.success && cart.data) dispatch({ type: "SET_CART", payload: cart.data });
        return { success: true, message: "Signed in." };
      },
      async signUp(payload) {
        ++sessionVersion.current;
        const version = sessionVersion.current;
        const result = await authApi.signUp({
          ...payload,
          confirm_password: payload.password,
        });
        setAuthReady(true);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }

        dispatch({ type: "SET_USERS", payload: [result.data] });
        dispatch({ type: "SIGN_IN", payload: result.data.id });
        const cart = await authApi.getCart();
        if (version === sessionVersion.current && cart.success && cart.data) dispatch({ type: "SET_CART", payload: cart.data });
        return { success: true, message: "Account created." };
      },
      async signOut() {
        ++sessionVersion.current;
        const version = sessionVersion.current;
        const result = await authApi.signOut();
        if (version !== sessionVersion.current) return;
        setAuthReady(true);
        if (!result.success) throw new Error(result.message);
        dispatch({ type: "SIGN_OUT" });
        const cart = await authApi.getCart();
        if (version === sessionVersion.current && cart.success && cart.data) dispatch({ type: "SET_CART", payload: cart.data });
      },
      async addToCart(productId) {
        const version = sessionVersion.current;
        const result = await authApi.addCartItem(productId);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_CART", payload: result.data });
        return { success: true, message: "Added to cart." };
      },
      async removeFromCart(productId) {
        const version = sessionVersion.current;
        const result = await authApi.removeCartItem(productId);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
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
        const version = sessionVersion.current;
        const result = await authApi.updateAddress(address);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_USERS", payload: [result.data] });
        return { success: true, message: "Shipping address saved." };
      },
      async setPaymentMethod(method) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }
        const version = sessionVersion.current;
        const result = await authApi.updatePaymentMethod({ type: method });
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_USERS", payload: [result.data] });
        return { success: true, message: "Payment method saved." };
      },
      async updateProfile(payload) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }
        const version = sessionVersion.current;
        const result = await authApi.updateProfile(payload);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_USERS", payload: [result.data] });
        return { success: true, message: "Profile updated." };
      },
      async placeOrder() {
        const version = sessionVersion.current;
        const result = await authApi.createOrder();
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({ type: "SET_ORDERS", payload: [result.data] });
        dispatch({ type: "SET_CART", payload: calcCart([]) });
        return { success: true, message: "Order created.", orderId: result.data.id };
      },
      async markOrderPaid(orderId) {
        const version = sessionVersion.current;
        const result = await authApi.markOrderPaid(orderId);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({
          type: "SET_ORDERS",
          payload: [result.data],
        });
        return { success: true, message: "Order marked as paid." };
      },
      async markOrderDelivered(orderId) {
        const version = sessionVersion.current;
        const result = await authApi.markOrderDelivered(orderId);
        if (version !== sessionVersion.current) return { success: false, message: "Session changed. Please retry." };
        if (!result.success || !result.data) {
          return { success: false, message: result.message };
        }
        dispatch({
          type: "SET_ORDERS",
          payload: state.orders.map((item) => (item.id === orderId ? result.data! : item)),
        });
        return { success: true, message: "Order marked as delivered." };
      },
      syncProducts,
      syncReviews,
    };
  }, [authReady, currentUser, state, syncProducts, syncReviews]);

  return <StoreContext.Provider value={value}>{children}</StoreContext.Provider>;
}

export function useStore() {
  const context = useContext(StoreContext);
  if (!context) {
    throw new Error("useStore must be used within StoreProvider");
  }
  return context;
}
