import { EllipsisVertical, MenuIcon, SearchIcon, ShoppingCart, Sun, UserIcon } from "lucide-react";
import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import * as api from "@/lib/api";
import { APP_NAME } from "@/lib/utils";

type CategoryOption = {
  category: string;
  count: number;
};

export function Header() {
  const navigate = useNavigate();
  const { currentUser, signOut, state } = useStore();
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("all");
  const [categories, setCategories] = useState<CategoryOption[]>([]);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [categoryMenuOpen, setCategoryMenuOpen] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement | null>(null);
  const categoryMenuRef = useRef<HTMLDivElement | null>(null);
  const mobileMenuRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadCategories() {
      const result = await api.getProductCategories();
      if (!cancelled && result.success && result.data) {
        setCategories(result.data);
      }
    }

    void loadCategories();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    function onPointerDown(event: MouseEvent) {
      const target = event.target as Node;
      if (!userMenuRef.current?.contains(target)) {
        setUserMenuOpen(false);
      }
      if (!categoryMenuRef.current?.contains(target)) {
        setCategoryMenuOpen(false);
      }
      if (!mobileMenuRef.current?.contains(target)) {
        setMobileMenuOpen(false);
      }
    }

    if (userMenuOpen || categoryMenuOpen || mobileMenuOpen) {
      document.addEventListener("mousedown", onPointerDown);
    }

    return () => {
      document.removeEventListener("mousedown", onPointerDown);
    };
  }, [categoryMenuOpen, mobileMenuOpen, userMenuOpen]);

  const cartCount = useMemo(
    () => state.cart.items.reduce((sum, item) => sum + item.qty, 0),
    [state.cart.items],
  );

  const firstInitial = currentUser?.name?.charAt(0).toUpperCase() ?? "U";

  function onSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const params = new URLSearchParams();
    const trimmed = query.trim();
    if (trimmed) params.set("q", trimmed);
    if (category !== "all") params.set("category", category);

    const suffix = params.toString();
    navigate(suffix ? `/search?${suffix}` : "/search");
  }

  async function onSignOut() {
    await signOut();
    setUserMenuOpen(false);
    setMobileMenuOpen(false);
    navigate("/");
  }

  return (
    <header className="w-full border-b">
      <div className="wrapper flex-between">
        <div className="flex-start" ref={categoryMenuRef}>
          <button
            type="button"
            className="inline-flex size-9 items-center justify-center gap-2 whitespace-nowrap rounded-md border bg-background px-3 text-sm font-medium shadow-xs transition-all hover:bg-accent hover:text-accent-foreground"
            aria-label="Open categories"
            onClick={() => setCategoryMenuOpen((current) => !current)}
          >
            <MenuIcon className="size-4" />
          </button>
          {categoryMenuOpen && (
            <div className="absolute left-5 top-20 z-30 h-full max-h-[70vh] w-full max-w-sm overflow-auto rounded-2xl border bg-white p-4 shadow-xl">
              <div className="text-lg font-semibold">Select a category</div>
              <div className="mt-4 space-y-1">
                {categories.map((item) => (
                  <button
                    key={item.category}
                    type="button"
                    className="flex w-full items-center justify-start gap-2 rounded-md px-3 py-2 text-sm hover:bg-accent"
                    onClick={() => {
                      setCategoryMenuOpen(false);
                      navigate(`/search?category=${encodeURIComponent(item.category)}`);
                    }}
                  >
                    {item.category} ({item.count})
                  </button>
                ))}
              </div>
            </div>
          )}
          <Link to="/" className="flex-start ml-4">
            <img src="/images/logo.svg" alt={`${APP_NAME} logo`} className="h-12 w-12" />
            <span className="ml-3 hidden text-2xl font-bold lg:block">{APP_NAME}</span>
          </Link>
        </div>

        <div className="hidden md:block">
          <form onSubmit={onSearchSubmit}>
            <div className="flex w-full max-w-sm items-center space-x-2">
              <select
                value={category}
                onChange={(event) => setCategory(event.target.value)}
                className="h-9 w-[180px] rounded-md border bg-background px-3 text-sm shadow-xs outline-none"
                aria-label="Category"
              >
                <option value="all">All</option>
                {categories.map((item) => (
                  <option key={item.category} value={item.category}>
                    {item.category}
                  </option>
                ))}
              </select>
              <input
                className="h-9 md:w-[100px] lg:w-[300px] rounded-md border bg-transparent px-3 py-1 text-base shadow-xs outline-none"
                placeholder="Search..."
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
              <button
                type="submit"
                className="inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-xs transition-all hover:bg-primary/90"
                aria-label="Search"
              >
                <SearchIcon className="size-4" />
              </button>
            </div>
          </form>
        </div>

        <div className="flex justify-end gap-3">
          <nav className="hidden w-full max-w-xs gap-1 md:flex">
            <button
              type="button"
              className="inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-md px-3 text-sm font-medium transition-all hover:bg-accent hover:text-accent-foreground"
              aria-label="Theme"
            >
              <Sun className="size-4" />
            </button>
            <Link
              to="/cart"
              className="inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-md px-3 text-sm font-medium transition-all hover:bg-accent hover:text-accent-foreground"
            >
              <ShoppingCart className="size-4" /> Cart
            </Link>

            <div className="relative flex items-center gap-2" ref={userMenuRef}>
              {!currentUser ? (
                <Link
                  to="/sign-in"
                  className="inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-xs transition-all hover:bg-primary/90"
                >
                  <UserIcon className="size-4" /> Sign In
                </Link>
              ) : (
                <>
                  <button
                    type="button"
                    className="relative ml-2 flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm"
                    onClick={() => setUserMenuOpen((current) => !current)}
                    aria-label="User menu"
                  >
                    {firstInitial}
                  </button>
                  {userMenuOpen && (
                    <div className="absolute right-0 top-12 z-30 w-56 rounded-md border bg-white p-1 shadow-md">
                      <div className="px-2 py-1.5">
                        <div className="text-sm font-medium leading-none">{currentUser.name}</div>
                        <div className="text-sm leading-none text-muted-foreground mt-1">{currentUser.email}</div>
                      </div>
                      <Link to="/user/profile" className="block w-full rounded-sm px-2 py-2 text-sm hover:bg-accent" onClick={() => setUserMenuOpen(false)}>
                        User Profile
                      </Link>
                      <Link to="/user/orders" className="block w-full rounded-sm px-2 py-2 text-sm hover:bg-accent" onClick={() => setUserMenuOpen(false)}>
                        Order History
                      </Link>
                      {currentUser.role === "admin" && (
                        <Link to="/admin/overview" className="block w-full rounded-sm px-2 py-2 text-sm hover:bg-accent" onClick={() => setUserMenuOpen(false)}>
                          Admin
                        </Link>
                      )}
                      <button
                        type="button"
                        className="flex w-full justify-start rounded-sm px-2 py-2 text-sm hover:bg-accent"
                        onClick={() => void onSignOut()}
                      >
                        Sign Out
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
          </nav>

          <nav className="md:hidden" ref={mobileMenuRef}>
            <button
              type="button"
              className="align-middle"
              onClick={() => setMobileMenuOpen((current) => !current)}
              aria-label="Open menu"
            >
              <EllipsisVertical className="size-5" />
            </button>

            {mobileMenuOpen && (
              <div className="absolute right-5 top-20 z-30 flex w-56 flex-col items-start rounded-2xl border bg-white p-4 shadow-xl">
                <button
                  type="button"
                  className="inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm hover:bg-accent"
                >
                  <Sun className="size-4" />
                </button>
                <Link to="/cart" className="inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm hover:bg-accent" onClick={() => setMobileMenuOpen(false)}>
                  <ShoppingCart className="size-4" /> Cart
                </Link>
                {!currentUser ? (
                  <Link
                    to="/sign-in"
                    className="inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm hover:bg-accent"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    <UserIcon className="size-4" /> Sign In
                  </Link>
                ) : (
                  <>
                    <div className="px-3 py-2 text-sm font-medium">{currentUser.name}</div>
                    <Link to="/user/profile" className="inline-flex h-9 items-center rounded-md px-3 text-sm hover:bg-accent" onClick={() => setMobileMenuOpen(false)}>
                      User Profile
                    </Link>
                    <Link to="/user/orders" className="inline-flex h-9 items-center rounded-md px-3 text-sm hover:bg-accent" onClick={() => setMobileMenuOpen(false)}>
                      Order History
                    </Link>
                    {currentUser.role === "admin" && (
                      <Link to="/admin/overview" className="inline-flex h-9 items-center rounded-md px-3 text-sm hover:bg-accent" onClick={() => setMobileMenuOpen(false)}>
                        Admin
                      </Link>
                    )}
                    <button type="button" className="inline-flex h-9 items-center rounded-md px-3 text-sm hover:bg-accent" onClick={() => void onSignOut()}>
                      Sign Out
                    </button>
                  </>
                )}
              </div>
            )}
          </nav>
        </div>
      </div>
    </header>
  );
}
