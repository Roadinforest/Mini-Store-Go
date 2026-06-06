import { ShoppingCart, UserCircle2 } from "lucide-react";
import { FormEvent, useState } from "react";
import { Link, NavLink, useNavigate } from "react-router-dom";
import { Button } from "@/components/common/Button";
import { useStore } from "@/app/store";
import { APP_NAME } from "@/lib/utils";

export function Header() {
  const navigate = useNavigate();
  const { currentUser, signOut, state } = useStore();
  const [query, setQuery] = useState("");

  function onSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = query.trim();
    navigate(trimmed ? `/search?q=${encodeURIComponent(trimmed)}` : "/search");
  }

  return (
    <header className="sticky top-0 z-20 border-b bg-white/95 backdrop-blur">
      <div className="wrapper flex flex-col gap-4 py-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-4">
          <Link to="/" className="flex items-center gap-3">
            <img src="/images/logo.svg" alt={`${APP_NAME} logo`} className="h-10 w-10" />
            <div>
              <div className="text-lg font-bold">{APP_NAME}</div>
              <div className="text-xs text-muted-foreground">Go + React mock frontend</div>
            </div>
          </Link>
          <nav className="hidden items-center gap-4 md:flex">
            <NavLink to="/" className="text-sm font-medium">
              Home
            </NavLink>
            <NavLink to="/search" className="text-sm font-medium">
              Search
            </NavLink>
          </nav>
        </div>

        <form onSubmit={onSearchSubmit} className="flex w-full max-w-xl gap-2">
          <input
            className="w-full rounded-md border px-3 py-2 text-sm"
            placeholder="Search products"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <Button type="submit">Search</Button>
        </form>

        <div className="flex items-center gap-3">
          <Link to="/cart" className="flex items-center gap-2 text-sm font-medium">
            <ShoppingCart className="h-4 w-4" />
            Cart ({state.cart.items.reduce((sum, item) => sum + item.qty, 0)})
          </Link>

          {currentUser ? (
            <div className="flex items-center gap-2">
              <UserCircle2 className="h-5 w-5" />
              <Link to="/user/profile" className="text-sm font-medium">
                {currentUser.name}
              </Link>
              <Link to="/user/orders" className="text-sm font-medium">
                Orders
              </Link>
              {currentUser.role === "admin" && (
                <>
                  <Link to="/admin/overview" className="text-sm font-medium">
                    Overview
                  </Link>
                  <Link to="/admin/products" className="text-sm font-medium">
                    Products
                  </Link>
                  <Link to="/admin/users" className="text-sm font-medium">
                    Users
                  </Link>
                  <Link to="/admin/orders" className="text-sm font-medium">
                    Admin Orders
                  </Link>
                </>
              )}
              <Button variant="outline" onClick={signOut}>
                Sign out
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <Link to="/sign-in" className="text-sm font-medium">
                Sign in
              </Link>
              <Link to="/sign-up" className="text-sm font-medium">
                Sign up
              </Link>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
