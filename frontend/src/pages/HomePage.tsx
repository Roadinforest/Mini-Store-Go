import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { HeroCarousel } from "@/components/product/HeroCarousel";
import { ProductList } from "@/components/product/ProductList";
import { useStore } from "@/app/store";
import * as api from "@/lib/api";
import type { Product } from "@/lib/types";

export function HomePage() {
  const { syncProducts } = useStore();
  const [featured, setFeatured] = useState<Product[]>([]);
  const [latest, setLatest] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const [featuredResult, latestResult] = await Promise.all([
        api.getFeaturedProducts(4),
        api.getLatestProducts(6),
      ]);
      if (cancelled) return;

      const nextFeatured = featuredResult.data ?? [];
      const nextLatest = latestResult.data ?? [];
      setFeatured(nextFeatured);
      setLatest(nextLatest);
      syncProducts([...nextFeatured, ...nextLatest]);
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <>
      <HeroCarousel products={featured} />
      {loading ? (
        <section className="my-10 rounded-3xl border p-6 text-sm text-muted-foreground">Loading products...</section>
      ) : (
        <ProductList products={latest} title="Newest Arrivals" limit={6} />
      )}
      <section className="grid gap-4 rounded-3xl border bg-slate-50 p-6 md:grid-cols-3">
        <div>
          <div className="text-sm font-semibold uppercase tracking-[0.3em] text-slate-500">Fast Migration</div>
          <div className="mt-2 text-xl font-semibold">Extracted from the original Next.js storefront</div>
        </div>
        <div className="text-sm text-slate-600">
          This mock frontend preserves the original store experience while removing
          Server Actions and backend coupling.
        </div>
        <div className="flex items-center justify-start md:justify-end">
          <Link to="/search" className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground">
            View all products
          </Link>
        </div>
      </section>
    </>
  );
}
