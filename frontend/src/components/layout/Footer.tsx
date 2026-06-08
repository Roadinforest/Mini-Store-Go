import { APP_NAME } from "@/lib/utils";

export function Footer() {
  const currentYear = new Date().getFullYear();
  return (
    <footer className="border-t">
      <div className="flex-center p-5">
        <span className="text-sm">
          &copy; {currentYear} {APP_NAME}. All Rights reserved.
        </span>
      </div>
    </footer>
  );
}
