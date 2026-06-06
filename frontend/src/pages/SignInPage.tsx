import { FormEvent, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";

export function SignInPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { signIn } = useStore();
  const [message, setMessage] = useState("Use admin@example.com / 123456 or user@example.com / 123456");

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const result = signIn({
      email: String(formData.get("email")),
      password: String(formData.get("password")),
    });
    setMessage(result.message);
    if (result.success) {
      navigate((location.state as { from?: string } | null)?.from ?? "/");
    }
  }

  return (
    <div className="mx-auto max-w-md rounded-3xl border p-6">
      <h1 className="h2-bold mb-4">Sign In</h1>
      <form onSubmit={onSubmit} className="grid gap-4">
        <input name="email" defaultValue="user@example.com" className="rounded-md border px-3 py-2" />
        <input name="password" type="password" defaultValue="123456" className="rounded-md border px-3 py-2" />
        <Button type="submit">Sign in</Button>
      </form>
      <div className="mt-4 text-sm text-muted-foreground">{message}</div>
      <div className="mt-4 text-sm">
        No account? <Link to="/sign-up">Create one</Link>
      </div>
    </div>
  );
}
