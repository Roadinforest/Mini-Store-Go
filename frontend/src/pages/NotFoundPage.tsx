import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="space-y-4 text-center">
      <h1 className="h1-bold">Page not found</h1>
      <p className="text-muted-foreground">The extracted frontend route does not exist.</p>
      <Link to="/">Back to home</Link>
    </div>
  );
}
