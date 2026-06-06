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
  addToCart: (productId: string) => Result;
  removeFromCart: (productId: string) => void;
  setShippingAddress: (address: ShippingAddress) => void;
  setPaymentMethod: (method: string) => void;
  updateProfile: (payload: ProfilePayload) => Result;
  placeOrder: () => { success: boolean; message: string; orderId?: string };
  markOrderPaid: (orderId: string) => { success: boolean; message: string };
  markOrderDelivered: (orderId: string) => void;
  saveProduct: (draft: ProductDraft) => void;
  deleteProduct: (productId: string) => void;
  updateUser: (userId: string, payload: Pick<User, "name" | "role">) => void;
  deleteUser: (userId: string) => void;
  upsertReview: (productId: string, payload: ReviewPayload) => Result;
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
      const result = await authApi.getCurrentUser();
      if (cancelled) return;

      if (result.success && result.data) {
        dispatch({ type: "SET_USERS", payload: upsertUser(state.users, result.data) });
        dispatch({ type: "SIGN_IN", payload: result.data.id });
      } else {
        dispatch({ type: "SIGN_OUT" });
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
      addToCart(productId) {
        const product = state.products.find((item) => item.id === productId);
        if (!product) return { success: false, message: "Product not found." };
        if (product.stock <= 0) return { success: false, message: "Not enough stock." };

        const existing = state.cart.items.find((item) => item.productId === productId);
        const nextQty = (existing?.qty ?? 0) + 1;
        if (nextQty > product.stock) {
          return { success: false, message: "Not enough stock." };
        }

        const nextItems = existing
          ? state.cart.items.map((item) =>
              item.productId === productId ? { ...item, qty: nextQty } : item,
            )
          : [
              ...state.cart.items,
              {
                productId: product.id,
                name: product.name,
                slug: product.slug,
                qty: 1,
                image: product.images[0],
                price: product.price,
              },
            ];

        dispatch({ type: "SET_CART", payload: calcCart(nextItems) });
        return { success: true, message: `${product.name} added to cart.` };
      },
      removeFromCart(productId) {
        const existing = state.cart.items.find((item) => item.productId === productId);
        if (!existing) return;

        const nextItems =
          existing.qty <= 1
            ? state.cart.items.filter((item) => item.productId !== productId)
            : state.cart.items.map((item) =>
                item.productId === productId ? { ...item, qty: item.qty - 1 } : item,
              );

        dispatch({ type: "SET_CART", payload: calcCart(nextItems) });
      },
      setShippingAddress(address) {
        if (!currentUser) return;
        const nextUsers = state.users.map((user) =>
          user.id === currentUser.id ? { ...user, address } : user,
        );
        dispatch({ type: "SET_USERS", payload: nextUsers });
      },
      setPaymentMethod(method) {
        if (!currentUser) return;
        const nextUsers = state.users.map((user) =>
          user.id === currentUser.id ? { ...user, paymentMethod: method } : user,
        );
        dispatch({ type: "SET_USERS", payload: nextUsers });
      },
      updateProfile(payload) {
        if (!currentUser) {
          return { success: false, message: "Sign in required." };
        }

        const nextUsers = state.users.map((user) =>
          user.id === currentUser.id
            ? { ...user, name: payload.name, email: payload.email }
            : user,
        );
        dispatch({ type: "SET_USERS", payload: nextUsers });
        return { success: true, message: "Profile updated." };
      },
      placeOrder() {
        if (!currentUser) {
          return { success: false, message: "Please sign in first." };
        }
        if (state.cart.items.length === 0) {
          return { success: false, message: "Cart is empty." };
        }
        if (!currentUser.address) {
          return { success: false, message: "Shipping address is required." };
        }
        if (!currentUser.paymentMethod) {
          return { success: false, message: "Payment method is required." };
        }

        const order: Order = {
          id: crypto.randomUUID(),
          userId: currentUser.id,
          shippingAddress: currentUser.address,
          paymentMethod: currentUser.paymentMethod,
          itemsPrice: state.cart.itemsPrice,
          shippingPrice: state.cart.shippingPrice,
          taxPrice: state.cart.taxPrice,
          totalPrice: state.cart.totalPrice,
          isPaid: false,
          paidAt: null,
          isDelivered: false,
          deliveredAt: null,
          createdAt: new Date().toISOString(),
          orderitems: state.cart.items,
        };

        dispatch({ type: "SET_ORDERS", payload: [order, ...state.orders] });
        dispatch({ type: "SET_CART", payload: calcCart([]) });
        return { success: true, message: "Order created.", orderId: order.id };
      },
      markOrderPaid(orderId) {
        const order = state.orders.find((item) => item.id === orderId);
        if (!order) return { success: false, message: "Order not found." };
        if (order.isPaid) return { success: false, message: "Order already paid." };

        for (const item of order.orderitems) {
          const product = state.products.find((candidate) => candidate.id === item.productId);
          if (!product || product.stock < item.qty) {
            return { success: false, message: `Not enough stock for ${item.name}.` };
          }
        }

        const nextProducts = state.products.map((product) => {
          const match = order.orderitems.find((item) => item.productId === product.id);
          if (!match) return product;
          return { ...product, stock: product.stock - match.qty };
        });
        const nextOrders = state.orders.map((item) =>
          item.id === orderId
            ? { ...item, isPaid: true, paidAt: new Date().toISOString() }
            : item,
        );

        dispatch({ type: "SET_PRODUCTS", payload: nextProducts });
        dispatch({ type: "SET_ORDERS", payload: nextOrders });
        return { success: true, message: "Order marked as paid." };
      },
      markOrderDelivered(orderId) {
        const nextOrders = state.orders.map((item) =>
          item.id === orderId
            ? { ...item, isDelivered: true, deliveredAt: new Date().toISOString() }
            : item,
        );
        dispatch({ type: "SET_ORDERS", payload: nextOrders });
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
