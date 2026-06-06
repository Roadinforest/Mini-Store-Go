import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ProductCard } from "@/components/product/ProductCard";
import { useStore } from "@/app/store";
import { getCategoryCounts } from "@/lib/utils";

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
  const { state } = useStore();
  const [params] = useSearchParams();
  const query = params.get("q") ?? "all";
  const category = params.get("category") ?? "all";
  const price = params.get("price") ?? "all";
  const rating = params.get("rating") ?? "all";
  const sort = params.get("sort") ?? "newest";
  const categories = getCategoryCounts(state.products);

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

  const filteredProducts = useMemo(() => {
    let products = [...state.products];

    if (query !== "all" && query.trim() !== "") {
      const lowered = query.toLowerCase();
      products = products.filter(
        (product) =>
          product.name.toLowerCase().includes(lowered) ||
          product.brand.toLowerCase().includes(lowered) ||
          product.description.toLowerCase().includes(lowered),
      );
    }

    if (category !== "all") {
      products = products.filter((product) => product.category === category);
    }

    if (price !== "all") {
      const [min, max] = price.split("-").map(Number);
      products = products.filter((product) => product.price >= min && product.price <= max);
    }

    if (rating !== "all") {
      products = products.filter((product) => product.rating >= Number(rating));
    }

    if (sort === "lowest") products.sort((a, b) => a.price - b.price);
    if (sort === "highest") products.sort((a, b) => b.price - a.price);
    if (sort === "rating") products.sort((a, b) => b.rating - a.rating);
    if (sort === "newest") products.sort((a, b) => b.createdAt.localeCompare(a.createdAt));

    return products;
  }, [category, price, query, rating, sort, state.products]);

  return (
    <div className="grid gap-8 md:grid-cols-5">
      <aside className="space-y-8">
        <div>
          <h3 className="mb-3 text-lg font-semibold">Department</h3>
          <div className="space-y-2 text-sm">
            <Link to={getFilterUrl({ category: "all" })}>Any</Link>
            {categories.map((item) => (
              <div key={item.category}>
                <Link to={getFilterUrl({ category: item.category })}>{item.category}</Link>
              </div>
            ))}
          </div>
        </div>

        <div>
          <h3 className="mb-3 text-lg font-semibold">Price</h3>
          <div className="space-y-2 text-sm">
            <Link to={getFilterUrl({ price: "all" })}>Any</Link>
            {prices.map((item) => (
              <div key={item.value}>
                <Link to={getFilterUrl({ price: item.value })}>{item.name}</Link>
              </div>
            ))}
          </div>
        </div>

        <div>
          <h3 className="mb-3 text-lg font-semibold">Customer Ratings</h3>
          <div className="space-y-2 text-sm">
            <Link to={getFilterUrl({ rating: "all" })}>Any</Link>
            {ratings.map((item) => (
              <div key={item}>
                <Link to={getFilterUrl({ rating: String(item) })}>{item} stars & up</Link>
              </div>
            ))}
          </div>
        </div>
      </aside>

      <section className="space-y-4 md:col-span-4">
        <div className="flex flex-col gap-4 rounded-2xl border p-4 md:flex-row md:items-center md:justify-between">
          <div className="text-sm text-muted-foreground">
            {query !== "all" && `Query: ${query} `}
            {category !== "all" && `Category: ${category} `}
            {price !== "all" && `Price: ${price} `}
            {rating !== "all" && `Rating: ${rating}+ `}
          </div>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span>Sort by</span>
            {sortOrders.map((item) => (
              <Link key={item} to={getFilterUrl({ sort: item })} className="font-medium">
                {item}
              </Link>
            ))}
            <Link to="/search" className="font-medium text-primary">
              Clear
            </Link>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {filteredProducts.map((product) => (
            <ProductCard key={product.id} product={product} />
          ))}
        </div>
      </section>
    </div>
  );
}
