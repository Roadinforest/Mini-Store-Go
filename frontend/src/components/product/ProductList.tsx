import { ProductCard } from "@/components/product/ProductCard";
import type { Product } from "@/lib/types";

export function ProductList({
  products,
  title,
  limit,
}: {
  products: Product[];
  title: string;
  limit?: number;
}) {
  const visible = limit ? products.slice(0, limit) : products;

  return (
    <section className="my-10">
      <h2 className="h2-bold mb-4">{title}</h2>
      {visible.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
          {visible.map((product) => (
            <ProductCard key={product.id} product={product} />
          ))}
        </div>
      ) : (
        <div>
          <p>No products found</p>
        </div>
      )}
    </section>
  );
}
