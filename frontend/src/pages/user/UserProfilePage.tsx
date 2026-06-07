import { FormEvent, useState } from "react";
import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";

export function UserProfilePage() {
  const { currentUser, updateProfile } = useStore();
  const [message, setMessage] = useState("");

  if (!currentUser) return null;

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const result = await updateProfile({
      name: String(formData.get("name")),
      email: String(formData.get("email")),
    });
    setMessage(result.message);
  }

  return (
    <div className="mx-auto max-w-xl rounded-3xl border p-6">
      <h1 className="h2-bold mb-4">My Profile</h1>
      <form onSubmit={onSubmit} className="grid gap-4">
        <input name="name" defaultValue={currentUser.name} className="rounded-md border px-3 py-2" />
        <input name="email" defaultValue={currentUser.email} className="rounded-md border px-3 py-2" />
        <Button type="submit">Save profile</Button>
      </form>
      {message && <div className="mt-4 text-sm text-muted-foreground">{message}</div>}
    </div>
  );
}
