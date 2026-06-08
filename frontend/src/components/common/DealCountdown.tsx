import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/common/Button";

const TARGET_DATE = new Date("2026-10-01T00:00:00");

function calculateTimeRemaining(targetDate: Date) {
  const currentTime = new Date();
  const timeDifference = Math.max(Number(targetDate) - Number(currentTime), 0);
  return {
    days: Math.floor(timeDifference / (1000 * 60 * 60 * 24)),
    hours: Math.floor((timeDifference % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)),
    minutes: Math.floor((timeDifference % (1000 * 60 * 60)) / (1000 * 60)),
    seconds: Math.floor((timeDifference % (1000 * 60)) / 1000),
  };
}

function StatBox({ label, value }: { label: string; value: number }) {
  return (
    <li className="w-full p-4 text-center">
      <p className="text-3xl font-bold">{value}</p>
      <p>{label}</p>
    </li>
  );
}

export function DealCountdown() {
  const [time, setTime] = useState<ReturnType<typeof calculateTimeRemaining>>();

  useEffect(() => {
    setTime(calculateTimeRemaining(TARGET_DATE));
    const timerInterval = window.setInterval(() => {
      const next = calculateTimeRemaining(TARGET_DATE);
      setTime(next);
      if (next.days === 0 && next.hours === 0 && next.minutes === 0 && next.seconds === 0) {
        window.clearInterval(timerInterval);
      }
    }, 1000);

    return () => window.clearInterval(timerInterval);
  }, []);

  if (!time) {
    return (
      <section className="my-20 grid grid-cols-1 md:grid-cols-2">
        <div className="flex flex-col justify-center gap-2">
          <h3 className="text-3xl font-bold">Loading Countdown...</h3>
        </div>
      </section>
    );
  }

  const ended = time.days === 0 && time.hours === 0 && time.minutes === 0 && time.seconds === 0;

  return (
    <section className="my-20 grid grid-cols-1 gap-8 md:grid-cols-2">
      <div className="flex flex-col justify-center gap-2">
        <h3 className="text-3xl font-bold">{ended ? "Deal Has Ended" : "Deal Of The Month"}</h3>
        <p>
          {ended
            ? "This deal is no longer available. Check out our latest promotions!"
            : "Get ready for a shopping experience like never before with our Deals of the Month. Every purchase comes with exclusive perks and offers."}
        </p>
        {!ended && (
          <ul className="grid grid-cols-4">
            <StatBox label="Days" value={time.days} />
            <StatBox label="Hours" value={time.hours} />
            <StatBox label="Minutes" value={time.minutes} />
            <StatBox label="Seconds" value={time.seconds} />
          </ul>
        )}
        <div className="text-center md:text-left">
          <Button asChild>
            <Link to="/search">View Products</Link>
          </Button>
        </div>
      </div>
      <div className="flex justify-center">
        <img src="/images/promo.jpg" alt="promotion" className="h-auto w-[300px]" />
      </div>
    </section>
  );
}
