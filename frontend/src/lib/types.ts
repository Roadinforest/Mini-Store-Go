export type UserRole = "admin" | "user";

export type Product = {
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
  numReviews: number;
  isFeatured: boolean;
  banner: string | null;
  createdAt: string;
};

export type ShippingAddress = {
  fullName: string;
  streetAddress: string;
  city: string;
  postalCode: string;
  country: string;
};

export type User = {
  id: string;
  name: string;
  email: string;
  password?: string;
  role: UserRole;
  address?: ShippingAddress;
  paymentMethod?: string;
  createdAt: string;
};

export type Review = {
  id: string;
  userId: string;
  productId: string;
  rating: number;
  title: string;
  description: string;
  isVerifiedPurchase: boolean;
  createdAt: string;
};

export type CartItem = {
  productId: string;
  name: string;
  slug: string;
  qty: number;
  image: string;
  price: number;
};

export type Cart = {
  items: CartItem[];
  itemsPrice: number;
  shippingPrice: number;
  taxPrice: number;
  totalPrice: number;
};

export type OrderItem = CartItem;

export type Order = {
  id: string;
  userId: string;
  shippingAddress: ShippingAddress;
  paymentMethod: string;
  itemsPrice: number;
  shippingPrice: number;
  taxPrice: number;
  totalPrice: number;
  isPaid: boolean;
  paidAt: string | null;
  isDelivered: boolean;
  deliveredAt: string | null;
  createdAt: string;
  orderitems: OrderItem[];
};

export type ProductDraft = Omit<Product, "id" | "rating" | "numReviews" | "createdAt"> & {
  id?: string;
  rating?: number;
  numReviews?: number;
  createdAt?: string;
};

export type AppState = {
  products: Product[];
  users: User[];
  reviews: Review[];
  orders: Order[];
  cart: Cart;
  currentUserId: string | null;
};
