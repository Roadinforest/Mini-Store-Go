import { useEffect, useState } from "react";

// Keep an already-open order page accurate when its payment window closes.
export function useOrderClock() {
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  return now;
}
