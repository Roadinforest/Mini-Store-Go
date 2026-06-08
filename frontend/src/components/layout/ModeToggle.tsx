import { MoonIcon, SunIcon, SunMoon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTheme, type ThemeMode } from "@/app/theme";

const items: Array<{ label: string; value: ThemeMode }> = [
  { label: "System", value: "system" },
  { label: "Light", value: "light" },
  { label: "Dark", value: "dark" },
];

export function ModeToggle() {
  const { theme, setTheme } = useTheme();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onPointerDown(event: MouseEvent) {
      if (!ref.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    if (open) {
      document.addEventListener("mousedown", onPointerDown);
    }

    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        className="inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-md px-3 text-sm font-medium transition-all hover:bg-accent hover:text-accent-foreground"
        onClick={() => setOpen((current) => !current)}
        aria-label="Appearance"
      >
        {theme === "system" ? (
          <SunMoon className="size-4" />
        ) : theme === "dark" ? (
          <MoonIcon className="size-4" />
        ) : (
          <SunIcon className="size-4" />
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-12 z-30 w-40 rounded-md border bg-popover p-1 text-popover-foreground shadow-md">
          <div className="px-2 py-1.5 text-sm font-medium">Appearance</div>
          <div className="my-1 h-px bg-border" />
          {items.map((item) => (
            <button
              key={item.value}
              type="button"
              className="flex w-full items-center justify-between rounded-sm px-2 py-2 text-sm hover:bg-accent"
              onClick={() => {
                setTheme(item.value);
                setOpen(false);
              }}
            >
              <span>{item.label}</span>
              {theme === item.value && <span className="text-xs text-muted-foreground">✓</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
