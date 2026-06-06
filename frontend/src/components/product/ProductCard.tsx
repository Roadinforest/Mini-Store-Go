import { Link } from "react-router-dom";
import { Rating } from "@/components/common/Rating";
import type { Product } from "@/lib/types";
import { formatCurrency } from "@/lib/utils";

export function ProductCard({ product }: { product: Product }) {
  return (
    <Link to={`/product/${product.slug}`} className="overflow-hidden rounded-2xl border bg-card transition hover:-translate-y-1 hover:shadow-lg">
      <img src={product.images[0]} alt={product.name} className="aspect-square w-full object-cover" />
      <div className="grid gap-3 p-4">
        <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">{product.brand}</div>
        <div className="font-semibold">{product.name}</div>
        <div className="flex-between gap-3">
          <Rating value={product.rating} />
          <div className={product.stock > 0 ? "font-semibold" : "font-semibold text-red-600"}>
            {product.stock > 0 ? formatCurrency(product.price) : "Out Of Stock"}
          </div>
        </div>
      </div>
    </Link>
  );
}
