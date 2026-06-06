import { Outlet } from "react-router-dom";
import { Footer } from "@/components/layout/Footer";
import { Header } from "@/components/layout/Header";

export function AppShell() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <Header />
      <main className="wrapper py-8">
        <Outlet />
      </main>
      <Footer />
    </div>
  );
}
