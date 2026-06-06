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
    <Link
      to={`/product/${product.slug}`}
      className="relative mb-10 block overflow-hidden rounded-3xl border bg-slate-900"
    >
      <img src={product.banner ?? product.images[0]} alt={product.name} className="h-[340px] w-full object-cover opacity-70" />
      <div className="absolute inset-0 bg-gradient-to-r from-slate-950 via-slate-900/40 to-transparent" />
      <div className="absolute inset-0 flex flex-col justify-end p-8 text-white">
        <div className="mb-2 text-sm uppercase tracking-[0.3em] text-slate-200">Featured Drop</div>
        <div className="max-w-xl text-3xl font-bold md:text-5xl">{product.name}</div>
        <div className="mt-3 max-w-lg text-sm text-slate-200">{product.description}</div>
      </div>
    </Link>
  );
}
