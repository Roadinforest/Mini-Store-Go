import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";

export function SignUpPage() {
  const navigate = useNavigate();
  const { signUp } = useStore();
  const [message, setMessage] = useState("");

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const password = String(formData.get("password"));
    const confirmPassword = String(formData.get("confirmPassword"));

    if (password !== confirmPassword) {
      setMessage("Passwords do not match.");
      return;
    }

    const result = await signUp({
      name: String(formData.get("name")),
      email: String(formData.get("email")),
      password,
    });
    setMessage(result.message);
    if (result.success) navigate("/");
  }

  return (
    <div className="mx-auto max-w-md rounded-3xl border p-6">
      <h1 className="h2-bold mb-4">Create account</h1>
      <form onSubmit={onSubmit} className="grid gap-4">
        <input name="name" placeholder="Name" className="rounded-md border px-3 py-2" required />
        <input name="email" placeholder="Email" className="rounded-md border px-3 py-2" required />
        <input name="password" type="password" placeholder="Password" className="rounded-md border px-3 py-2" required />
        <input name="confirmPassword" type="password" placeholder="Confirm password" className="rounded-md border px-3 py-2" required />
        <Button type="submit">Sign up</Button>
      </form>
      {message && <div className="mt-4 text-sm text-muted-foreground">{message}</div>}
    </div>
  );
}
