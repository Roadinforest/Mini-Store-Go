import clsx from "clsx";
import { NavLink } from "react-router-dom";

export function SectionNav({
  links,
  className,
}: {
  links: Array<{ title: string; href: string }>;
  className?: string;
}) {
  return (
    <nav className={clsx("flex items-center space-x-4 lg:space-x-6", className)}>
      {links.map((item) => (
        <NavLink
          key={item.href}
          to={item.href}
          className={({ isActive }) =>
            clsx(
              "text-sm font-medium transition-colors hover:text-primary",
              isActive ? "" : "text-muted-foreground",
            )
          }
        >
          {item.title}
        </NavLink>
      ))}
    </nav>
  );
}
