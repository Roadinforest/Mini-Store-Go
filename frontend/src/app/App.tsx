import type { ReactElement } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { StoreProvider, useStore } from "@/app/store";
import { AppShell } from "@/components/layout/AppShell";
import { AdminOrdersPage } from "@/pages/admin/AdminOrdersPage";
import { AdminOverviewPage } from "@/pages/admin/AdminOverviewPage";
import { AdminProductsPage } from "@/pages/admin/AdminProductsPage";
import { AdminUsersPage } from "@/pages/admin/AdminUsersPage";
import { CartPage } from "@/pages/CartPage";
import { HomePage } from "@/pages/HomePage";
import { NotFoundPage } from "@/pages/NotFoundPage";
import { OrderDetailsPage } from "@/pages/OrderDetailsPage";
import { PaymentMethodPage } from "@/pages/PaymentMethodPage";
import { PlaceOrderPage } from "@/pages/PlaceOrderPage";
import { ProductPage } from "@/pages/ProductPage";
import { SearchPage } from "@/pages/SearchPage";
import { ShippingAddressPage } from "@/pages/ShippingAddressPage";
import { SignInPage } from "@/pages/SignInPage";
import { SignUpPage } from "@/pages/SignUpPage";
import { UnauthorizedPage } from "@/pages/UnauthorizedPage";
import { UserOrdersPage } from "@/pages/user/UserOrdersPage";
import { UserProfilePage } from "@/pages/user/UserProfilePage";

function RequireAuth({ children }: { children: ReactElement }) {
  const location = useLocation();
  const { authReady, currentUser } = useStore();
  if (!authReady) {
    return <div className="wrapper py-10 text-sm text-muted-foreground">Checking session...</div>;
  }
  if (!currentUser) {
    return <Navigate to="/sign-in" replace state={{ from: location.pathname }} />;
  }
  return children;
}

function RequireAdmin({ children }: { children: ReactElement }) {
  const { authReady, currentUser } = useStore();
  if (!authReady) {
    return <div className="wrapper py-10 text-sm text-muted-foreground">Checking session...</div>;
  }
  if (!currentUser) return <Navigate to="/sign-in" replace state={{ from: "/admin/overview" }} />;
  if (currentUser.role !== "admin") return <Navigate to="/unauthorized" replace />;
  return children;
}

function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/product/:slug" element={<ProductPage />} />
        <Route path="/cart" element={<CartPage />} />
        <Route path="/sign-in" element={<SignInPage />} />
        <Route path="/sign-up" element={<SignUpPage />} />
        <Route
          path="/shipping-address"
          element={
            <RequireAuth>
              <ShippingAddressPage />
            </RequireAuth>
          }
        />
        <Route
          path="/payment-method"
          element={
            <RequireAuth>
              <PaymentMethodPage />
            </RequireAuth>
          }
        />
        <Route
          path="/place-order"
          element={
            <RequireAuth>
              <PlaceOrderPage />
            </RequireAuth>
          }
        />
        <Route
          path="/order/:id"
          element={
            <RequireAuth>
              <OrderDetailsPage />
            </RequireAuth>
          }
        />
        <Route
          path="/user/profile"
          element={
            <RequireAuth>
              <UserProfilePage />
            </RequireAuth>
          }
        />
        <Route
          path="/user/orders"
          element={
            <RequireAuth>
              <UserOrdersPage />
            </RequireAuth>
          }
        />
        <Route
          path="/admin/overview"
          element={
            <RequireAdmin>
              <AdminOverviewPage />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/products"
          element={
            <RequireAdmin>
              <AdminProductsPage />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/users"
          element={
            <RequireAdmin>
              <AdminUsersPage />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/orders"
          element={
            <RequireAdmin>
              <AdminOrdersPage />
            </RequireAdmin>
          }
        />
        <Route path="/unauthorized" element={<UnauthorizedPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}

export function App() {
  return (
    <StoreProvider>
      <AppRoutes />
    </StoreProvider>
  );
}
