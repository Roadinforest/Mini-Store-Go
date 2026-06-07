import { useEffect, useState } from "react";
import { Button } from "@/components/common/Button";
import * as api from "@/lib/api";
import type { User } from "@/lib/types";

export function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const result = await api.getAdminUsers({ page: 1, limit: 100 });
      if (cancelled) return;
      setUsers(result.data?.items ?? []);
      setLoading(false);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  async function onToggleRole(user: User) {
    const nextRole = user.role === "admin" ? "user" : "admin";
    const result = await api.updateAdminUser(user.id, {
      name: user.name,
      email: user.email,
      role: nextRole,
    });
    setMessage(result.message);
    if (result.success && result.data) {
      setUsers((current) => current.map((item) => (item.id === user.id ? result.data! : item)));
    }
  }

  async function onDelete(userID: string) {
    const result = await api.deleteAdminUser(userID);
    setMessage(result.message);
    if (result.success) {
      setUsers((current) => current.filter((user) => user.id !== userID));
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="h2-bold">Users</h1>
      <div className="overflow-x-auto rounded-3xl border">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left">
            <tr>
              <th className="p-4">Name</th>
              <th className="p-4">Email</th>
              <th className="p-4">Role</th>
              <th className="p-4">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="p-4 text-muted-foreground" colSpan={4}>
                  Loading users...
                </td>
              </tr>
            ) : (
              users.map((user) => (
                <tr key={user.id} className="border-t">
                  <td className="p-4">{user.name}</td>
                  <td className="p-4">{user.email}</td>
                  <td className="p-4">{user.role}</td>
                  <td className="p-4">
                    <div className="flex gap-2">
                      <Button variant="outline" onClick={() => void onToggleRole(user)}>
                        Toggle role
                      </Button>
                      <Button variant="danger" onClick={() => void onDelete(user.id)}>
                        Delete
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {message && <div className="text-sm text-muted-foreground">{message}</div>}
    </div>
  );
}
