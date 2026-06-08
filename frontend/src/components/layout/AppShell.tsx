import { Outlet, useLocation } from "react-router-dom";
import { Footer } from "@/components/layout/Footer";
import { Header } from "@/components/layout/Header";
import { SectionNav } from "@/components/layout/SectionNav";

const adminLinks = [
  { title: "Overview", href: "/admin/overview" },
  { title: "Products", href: "/admin/products" },
  { title: "Orders", href: "/admin/orders" },
  { title: "Users", href: "/admin/users" },
];

const userLinks = [
  { title: "Profile", href: "/user/profile" },
  { title: "Orders", href: "/user/orders" },
];

export function AppShell() {
  const location = useLocation();
  const isAdmin = location.pathname.startsWith("/admin");
  const isUser = location.pathname.startsWith("/user");

  return (
    <div className="min-h-screen bg-background text-foreground">
      <Header />
      {(isAdmin || isUser) && (
        <div className="container mx-auto border-b">
          <div className="flex h-16 items-center px-4">
            <SectionNav links={isAdmin ? adminLinks : userLinks} className="mx-0" />
          </div>
        </div>
      )}
      <main className={isAdmin || isUser ? "container mx-auto flex-1 space-y-4 p-8 pt-6" : "wrapper py-8"}>
        <Outlet />
      </main>
      <Footer />
    </div>
  );
}
