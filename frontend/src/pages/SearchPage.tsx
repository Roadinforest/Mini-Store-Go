import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ProductCard } from "@/components/product/ProductCard";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import * as api from "@/lib/api";
import type { Product } from "@/lib/types";

const prices = [
  { name: "$1 to $50", value: "1-50" },
  { name: "$51 to $100", value: "51-100" },
  { name: "$101 to $200", value: "101-200" },
  { name: "$201 to $500", value: "201-500" },
  { name: "$501 to $1000", value: "501-1000" },
];

const ratings = [4, 3, 2, 1];
const sortOrders = ["newest", "lowest", "highest", "rating"];

export function SearchPage() {
  const { syncProducts } = useStore();
  const [params] = useSearchParams();
  const query = params.get("q") ?? "all";
  const category = params.get("category") ?? "all";
  const price = params.get("price") ?? "all";
  const rating = params.get("rating") ?? "all";
  const sort = params.get("sort") ?? "newest";
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Array<{ category: string; count: number }>>([]);
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState("");

  function getFilterUrl(next: Record<string, string>) {
    const search = new URLSearchParams({
      q: query,
      category,
      price,
      rating,
      sort,
      ...next,
    });
    return `/search?${search.toString()}`;
  }

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      const [productsResult, categoriesResult] = await Promise.all([
        api.getProducts({
          page: 1,
          limit: 60,
          q: query,
          category,
          price,
          rating,
          sort,
        }),
        api.getProductCategories(),
      ]);
      if (cancelled) return;

      const nextProducts = productsResult.data?.items ?? [];
      setProducts(nextProducts);
      setCategories(categoriesResult.data ?? []);
      syncProducts(nextProducts);
      setSummary(
        productsResult.success && productsResult.data
          ? `${productsResult.data.meta.total} products found`
          : productsResult.message,
      );
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [category, price, query, rating, sort]);

  return (
    <div className="grid md:grid-cols-5 md:gap-5">
      <aside className="filter-links">
        <div>
          <div className="mb-2 mt-3 text-xl">Department</div>
          <div>
            <ul className="space-y-1">
              <li>
                <Link className={(category === "all" || category === "") ? "font-bold" : ""} to={getFilterUrl({ category: "all" })}>Any</Link>
              </li>
              {categories.map((item) => (
                <li key={item.category}>
                  <Link className={category === item.category ? "font-bold" : ""} to={getFilterUrl({ category: item.category })}>
                    {item.category}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div>
          <div className="mb-2 mt-8 text-xl">Price</div>
          <div>
            <ul className="space-y-1">
              <li>
                <Link className={price === "all" ? "font-bold" : ""} to={getFilterUrl({ price: "all" })}>Any</Link>
              </li>
              {prices.map((item) => (
                <li key={item.value}>
                  <Link className={price === item.value ? "font-bold" : ""} to={getFilterUrl({ price: item.value })}>
                    {item.name}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div>
          <div className="mb-2 mt-8 text-xl">Customer Ratings</div>
          <div>
            <ul className="space-y-1">
              <li>
                <Link className={rating === "all" ? "font-bold" : ""} to={getFilterUrl({ rating: "all" })}>Any</Link>
              </li>
              {ratings.map((item) => (
                <li key={item}>
                  <Link className={rating === String(item) ? "font-bold" : ""} to={getFilterUrl({ rating: String(item) })}>
                    {item} stars & up
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </aside>

      <section className="space-y-4 md:col-span-4">
        <div className="my-4 flex-between flex-col md:flex-row">
          <div className="flex items-center">
            <p>
              {query !== "all" && query !== "" && `Query: ${query}`}
              {category !== "all" && category !== "" && `Category: ${category}`}
              {price !== "all" && ` Price: ${price}`}
              {rating !== "all" && ` Rating: ${rating} stars & up`}
              &nbsp;
            </p>
            {(query !== "all" && query !== "") || (category !== "all" && category !== "") || rating !== "all" || price !== "all" ? (
              <div>
                <Button variant="outline" asChild to="/search">
                  Clear
                </Button>
              </div>
            ) : null}
          </div>
          <div>
            Sort by{" "}
            {sortOrders.map((item) => (
              <Link key={item} to={getFilterUrl({ sort: item })} className={`mx-2 ${sort === item ? "font-bold" : ""}`}>
                {item}
              </Link>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {loading ? (
            <div className="rounded-2xl border p-4 text-sm text-muted-foreground">Loading products...</div>
          ) : products.length === 0 ? (
            <div>No products found</div>
          ) : (
            products.map((product) => <ProductCard key={product.id} product={product} />)
          )}
        </div>
      </section>
    </div>
  );
}
