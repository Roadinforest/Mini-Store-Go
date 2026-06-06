import { FormEvent, useState } from "react";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";

export function AdminProductsPage() {
  const { state, deleteProduct, saveProduct } = useStore();
  const [message, setMessage] = useState("");

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    saveProduct({
      name: String(formData.get("name")),
      slug: String(formData.get("slug")),
      category: String(formData.get("category")),
      brand: String(formData.get("brand")),
      description: String(formData.get("description")),
      price: Number(formData.get("price")),
      stock: Number(formData.get("stock")),
      images: [String(formData.get("image"))],
      isFeatured: Boolean(formData.get("isFeatured")),
      banner: String(formData.get("banner")) || null,
    });
    setMessage("Product saved.");
    event.currentTarget.reset();
  }

  return (
    <div className="grid gap-8 lg:grid-cols-[1.2fr_0.8fr]">
      <div className="space-y-4">
        <h1 className="h2-bold">Products</h1>
        <div className="overflow-x-auto rounded-3xl border">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left">
              <tr>
                <th className="p-4">Name</th>
                <th className="p-4">Stock</th>
                <th className="p-4">Price</th>
                <th className="p-4">Actions</th>
              </tr>
            </thead>
            <tbody>
              {state.products.map((product) => (
                <tr key={product.id} className="border-t">
                  <td className="p-4">{product.name}</td>
                  <td className="p-4">{product.stock}</td>
                  <td className="p-4">${product.price.toFixed(2)}</td>
                  <td className="p-4">
                    <Button variant="outline" onClick={() => deleteProduct(product.id)}>
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <form onSubmit={onSubmit} className="grid gap-3 rounded-3xl border p-5">
        <h2 className="text-lg font-semibold">Create Product</h2>
        <input name="name" placeholder="Name" className="rounded-md border px-3 py-2" required />
        <input name="slug" placeholder="Slug" className="rounded-md border px-3 py-2" required />
        <input name="category" placeholder="Category" className="rounded-md border px-3 py-2" required />
        <input name="brand" placeholder="Brand" className="rounded-md border px-3 py-2" required />
        <input name="image" placeholder="/images/sample-products/p1-1.jpg" className="rounded-md border px-3 py-2" required />
        <input name="banner" placeholder="/images/banner-1.jpg" className="rounded-md border px-3 py-2" />
        <input name="price" type="number" step="0.01" placeholder="Price" className="rounded-md border px-3 py-2" required />
        <input name="stock" type="number" placeholder="Stock" className="rounded-md border px-3 py-2" required />
        <textarea name="description" placeholder="Description" className="min-h-28 rounded-md border px-3 py-2" required />
        <label className="flex items-center gap-2">
          <input type="checkbox" name="isFeatured" />
          <span className="text-sm">Featured</span>
        </label>
        <Button type="submit">Save product</Button>
        {message && <div className="text-sm text-muted-foreground">{message}</div>}
      </form>
    </div>
  );
}
