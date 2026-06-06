import { Link } from "react-router-dom";
import { HeroCarousel } from "@/components/product/HeroCarousel";
import { ProductList } from "@/components/product/ProductList";
import { useStore } from "@/app/store";

export function HomePage() {
  const { state } = useStore();
  const featured = state.products.filter((product) => product.isFeatured);
  const latest = [...state.products].sort((a, b) => b.createdAt.localeCompare(a.createdAt));

  return (
    <>
      <HeroCarousel products={featured} />
      <ProductList products={latest} title="Newest Arrivals" limit={6} />
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
