import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import type { Product } from "@/lib/types";

export function HeroCarousel({ products }: { products: Product[] }) {
  const [index, setIndex] = useState(0);

  useEffect(() => {
    if (products.length <= 1) return;
    const timer = window.setInterval(() => {
      setIndex((current) => (current + 1) % products.length);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [products.length]);

  if (products.length === 0) return null;

  const product = products[index];
  return (
    <Link to={`/product/${product.slug}`} className="mb-12 block">
      <div className="relative mx-auto">
        <img src={product.banner ?? product.images[0]} alt={product.name} className="h-auto w-full" />
        <div className="absolute inset-0 flex items-end justify-center">
          <h2 className="bg-gray-900 bg-opacity-50 px-2 text-2xl font-bold text-white">
            {product.name}
          </h2>
        </div>
      </div>
    </Link>
  );
}
