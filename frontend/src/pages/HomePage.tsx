import { useEffect, useState } from "react";
import { DealCountdown } from "@/components/common/DealCountdown";
import { IconBoxes } from "@/components/common/IconBoxes";
import { ViewAllProductsButton } from "@/components/common/ViewAllProductsButton";
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
      {featured.length > 0 && <HeroCarousel products={featured} />}
      {loading ? (
        <section className="my-10 rounded-3xl border p-6 text-sm text-muted-foreground">Loading products...</section>
      ) : (
        <ProductList products={latest} title="Newest Arrivals" limit={6} />
      )}
      <ViewAllProductsButton />
      <DealCountdown />
      <IconBoxes />
    </>
  );
}
