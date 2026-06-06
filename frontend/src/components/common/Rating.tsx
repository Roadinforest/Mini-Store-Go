import { Star } from "lucide-react";

export function Rating({ value }: { value: number }) {
  const fullStars = Math.round(value);
  return (
    <div className="flex items-center gap-1 text-amber-500">
      {Array.from({ length: 5 }).map((_, index) => (
        <Star
          key={index}
          className={`h-4 w-4 ${index < fullStars ? "fill-current" : ""}`}
        />
      ))}
      <span className="ml-1 text-sm text-muted-foreground">{value.toFixed(1)}</span>
    </div>
  );
}
