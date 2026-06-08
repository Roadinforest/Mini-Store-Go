import { Link } from "react-router-dom";
import { Rating } from "@/components/common/Rating";
import type { Product } from "@/lib/types";
import { formatCurrency } from "@/lib/utils";

export function ProductCard({ product }: { product: Product }) {
  return (
    <div className="w-full max-w-sm rounded-lg border bg-card">
      <Link to={`/product/${product.slug}`}>
        <div className="flex items-center justify-center p-0">
          <img src={product.images[0]} alt={product.name} className="h-[300px] w-[300px] object-cover" />
        </div>
        <div className="grid gap-4 p-4">
          <div className="text-xs">{product.brand}</div>
          <h2 className="text-sm font-medium">{product.name}</h2>
          <div className="flex-between gap-4">
            <Rating value={product.rating} />
            {product.stock > 0 ? (
              <div>{formatCurrency(product.price)}</div>
            ) : (
              <p className="text-destructive">Out Of Stock</p>
            )}
          </div>
        </div>
      </Link>
    </div>
  );
}
