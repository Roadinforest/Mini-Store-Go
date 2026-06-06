import { FormEvent, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";
import { Rating } from "@/components/common/Rating";
import { formatCurrency } from "@/lib/utils";

export function ProductPage() {
  const { slug } = useParams();
  const { state, currentUser, addToCart, upsertReview } = useStore();
  const [imageIndex, setImageIndex] = useState(0);
  const [message, setMessage] = useState("");

  const product = state.products.find((item) => item.slug === slug);
  const reviews = useMemo(
    () => state.reviews.filter((review) => review.productId === product?.id),
    [product?.id, state.reviews],
  );

  if (!product) {
    return <div>Product not found.</div>;
  }

  function submitReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!product) return;
    const formData = new FormData(event.currentTarget);
    const result = upsertReview(product.id, {
      rating: Number(formData.get("rating")),
      title: String(formData.get("title")),
      description: String(formData.get("description")),
    });
    setMessage(result.message);
    if (result.success) {
      event.currentTarget.reset();
    }
  }

  return (
    <div className="space-y-10">
      <section className="grid gap-8 md:grid-cols-5">
        <div className="md:col-span-2">
          <img src={product.images[imageIndex]} alt={product.name} className="w-full rounded-3xl border object-cover" />
          <div className="mt-4 flex gap-3">
            {product.images.map((image, index) => (
              <button key={image} onClick={() => setImageIndex(index)} className="overflow-hidden rounded-xl border">
                <img src={image} alt={`${product.name} ${index + 1}`} className="h-20 w-20 object-cover" />
              </button>
            ))}
          </div>
        </div>
        <div className="space-y-5 md:col-span-2">
          <div className="text-sm uppercase tracking-[0.2em] text-muted-foreground">
            {product.brand} · {product.category}
          </div>
          <h1 className="h1-bold">{product.name}</h1>
          <Rating value={product.rating} />
          <p className="text-sm text-muted-foreground">{product.numReviews} reviews</p>
          <div className="inline-flex rounded-full bg-green-100 px-5 py-2 font-semibold text-green-700">
            {formatCurrency(product.price)}
          </div>
          <p className="leading-7 text-muted-foreground">{product.description}</p>
        </div>
        <div className="rounded-3xl border p-5">
          <div className="flex-between py-3 text-sm">
            <span>Price</span>
            <span className="font-semibold">{formatCurrency(product.price)}</span>
          </div>
          <div className="flex-between py-3 text-sm">
            <span>Status</span>
            <span className={product.stock > 0 ? "font-semibold text-green-700" : "font-semibold text-red-600"}>
              {product.stock > 0 ? "In Stock" : "Out Of Stock"}
            </span>
          </div>
          <Button
            className="mt-4 w-full"
            disabled={product.stock <= 0}
            onClick={() => setMessage(addToCart(product.id).message)}
          >
            Add to cart
          </Button>
          {message && <div className="mt-3 text-sm text-muted-foreground">{message}</div>}
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-2">
        <div>
          <h2 className="h2-bold mb-5">Customer Reviews</h2>
          <div className="space-y-4">
            {reviews.map((review) => {
              const author = state.users.find((user) => user.id === review.userId);
              return (
                <div key={review.id} className="rounded-2xl border p-4">
                  <div className="flex-between gap-3">
                    <div>
                      <div className="font-semibold">{review.title}</div>
                      <div className="text-sm text-muted-foreground">{author?.name ?? "Unknown user"}</div>
                    </div>
                    <Rating value={review.rating} />
                  </div>
                  <p className="mt-3 text-sm leading-6 text-muted-foreground">{review.description}</p>
                </div>
              );
            })}
          </div>
        </div>

        <div>
          <h2 className="h2-bold mb-5">Write a review</h2>
          {currentUser ? (
            <form onSubmit={submitReview} className="grid gap-4 rounded-2xl border p-5">
              <select name="rating" className="rounded-md border px-3 py-2" defaultValue="5">
                {[5, 4, 3, 2, 1].map((value) => (
                  <option key={value} value={value}>
                    {value} stars
                  </option>
                ))}
              </select>
              <input name="title" className="rounded-md border px-3 py-2" placeholder="Title" required />
              <textarea
                name="description"
                className="min-h-32 rounded-md border px-3 py-2"
                placeholder="Description"
                required
              />
              <Button type="submit">Save review</Button>
              {message && <div className="text-sm text-muted-foreground">{message}</div>}
            </form>
          ) : (
            <div className="rounded-2xl border p-5 text-sm text-muted-foreground">
              Sign in to write a review.
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
