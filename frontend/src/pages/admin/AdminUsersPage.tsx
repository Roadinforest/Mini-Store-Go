import { useStore } from "@/app/store";
import { Button } from "@/components/common/Button";

export function AdminUsersPage() {
  const { state, updateUser, deleteUser } = useStore();

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
            {state.users.map((user) => (
              <tr key={user.id} className="border-t">
                <td className="p-4">{user.name}</td>
                <td className="p-4">{user.email}</td>
                <td className="p-4">{user.role}</td>
                <td className="p-4">
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      onClick={() => updateUser(user.id, { name: user.name, role: user.role === "admin" ? "user" : "admin" })}
                    >
                      Toggle role
                    </Button>
                    <Button variant="danger" onClick={() => deleteUser(user.id)}>
                      Delete
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
