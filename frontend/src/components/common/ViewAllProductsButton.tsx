import { Link } from "react-router-dom";
import { Button } from "@/components/common/Button";

export function ViewAllProductsButton() {
  return (
    <div className="my-8 flex items-center justify-center">
      <Button asChild className="px-8 py-4 text-lg font-semibold">
        <Link to="/search">View All Products</Link>
      </Button>
    </div>
  );
}
